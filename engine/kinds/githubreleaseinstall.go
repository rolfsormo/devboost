package kinds

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rolfsormo/devboost/engine"
)

// GitHubReleaseInstall declares "download this tool's latest GitHub
// release binary and extract it into devboost's managed bin dir" — for
// tools with neither an apt package nor an official install script
// (lazygit, procs — confirmed via research: lazygit has no curl|sh
// installer at all; procs's README only lists package managers not
// available on bare Ubuntu/Debian, plus `cargo install`). This is the
// same "no official script exists, so use the vendor's own release
// artifact directly" fallback GitHub itself is the canonical
// distribution point for, one step more manual than VendorInstall but
// still not devboost inventing its own packaging.
//
// AssetNameFor must return the exact release-asset filename for a given
// (goos, goarch) pair — different projects name these differently
// (lazygit: "lazygit_<version>_linux_x86_64.tar.gz"; procs:
// "procs-v<version>-x86_64-linux.zip"), so this is left to the caller
// rather than guessed generically.
type GitHubReleaseInstall struct {
	BinaryName      string // e.g. "lazygit" — used for the idempotency check and the extracted file to look for
	Repo            string // "owner/repo", e.g. "jesseduffield/lazygit"
	AssetNameFor    func(version, goos, goarch string) (assetName string, archive ArchiveKind)
	BinaryInArchive string // path of the binary inside the archive, e.g. "lazygit" (usually just BinaryName)
}

// ArchiveKind is exported so callers outside this package (module code
// building an AssetNameFor closure) can name the archive format their
// tool's release actually ships without needing an unexported type from
// this package.
type ArchiveKind int

const (
	ArchiveTarGz ArchiveKind = iota
	ArchiveZip
)

func (g GitHubReleaseInstall) Diff() (*engine.PendingOp, error) {
	found, err := BinaryAvailable(g.BinaryName)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, nil
	}

	return &engine.PendingOp{
		Description: fmt.Sprintf("install %s from its latest GitHub release", g.BinaryName),
		Execute: func() error {
			return g.install()
		},
	}, nil
}

func (g GitHubReleaseInstall) install() error {
	dir, err := ManagedBinDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	version, err := latestReleaseTag(g.Repo)
	if err != nil {
		return fmt.Errorf("resolve latest release for %s: %w", g.Repo, err)
	}
	// Asset filenames are conventionally version-without-leading-v even
	// when the git tag has one (confirmed for both lazygit and procs) —
	// callers' AssetNameFor receives the raw tag and decide themselves.
	assetName, kind := g.AssetNameFor(version, runtime.GOOS, runtime.GOARCH)

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", g.Repo, version, assetName)
	body, err := fetchBody(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", assetName, err)
	}
	defer body.Close()

	tmp, err := os.CreateTemp("", g.BinaryName+"-download-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		return fmt.Errorf("save %s: %w", assetName, err)
	}
	tmp.Close()

	binaryInArchive := g.BinaryInArchive
	if binaryInArchive == "" {
		binaryInArchive = g.BinaryName
	}

	dest := filepath.Join(dir, g.BinaryName)
	switch kind {
	case ArchiveTarGz:
		err = extractFromTarGz(tmp.Name(), binaryInArchive, dest)
	case ArchiveZip:
		err = extractFromZip(tmp.Name(), binaryInArchive, dest)
	default:
		err = fmt.Errorf("unknown archive kind")
	}
	if err != nil {
		return fmt.Errorf("extract %s from %s: %w", binaryInArchive, assetName, err)
	}
	return os.Chmod(dest, 0o755)
}

// latestReleaseTag resolves a GitHub repo's latest release tag (e.g.
// "v0.64.0") via the public API — unauthenticated, subject to GitHub's
// standard 60 req/hr per-IP rate limit, acceptable for a once-per-apply
// lookup per tool.
func latestReleaseTag(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from GitHub API", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("no tag_name in GitHub API response")
	}
	return payload.TagName, nil
}

func extractFromTarGz(archivePath, wantName, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s not found in archive", wantName)
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) != wantName || hdr.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, tr)
		return err
	}
}

func extractFromZip(archivePath, wantName, dest string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) != wantName || strings.HasSuffix(f.Name, "/") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, rc)
		return err
	}
	return fmt.Errorf("%s not found in archive", wantName)
}
