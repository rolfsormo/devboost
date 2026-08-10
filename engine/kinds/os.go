package kinds

import (
	"os"
	"runtime"
)

// OS identifies the target platform, mirroring the bash tool's DB_OS
// values exactly (darwin, linux-ubuntu, linux-fedora, linux-arch, other).
type OS string

const (
	OSDarwin      OS = "darwin"
	OSLinuxUbuntu OS = "linux-ubuntu"
	OSLinuxFedora OS = "linux-fedora"
	OSLinuxArch   OS = "linux-arch"
	OSOther       OS = "other"
)

// DetectOS ports db_detect_os: darwin via GOOS, Linux distro via the same
// marker files the bash version checks.
func DetectOS() OS {
	if runtime.GOOS == "darwin" {
		return OSDarwin
	}
	if fileExists("/etc/debian_version") {
		return OSLinuxUbuntu
	}
	if fileExists("/etc/fedora-release") {
		return OSLinuxFedora
	}
	if fileExists("/etc/arch-release") {
		return OSLinuxArch
	}
	return OSOther
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
