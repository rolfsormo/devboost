# Testing devboost

This directory contains test scripts and Docker configurations for testing devboost on different platforms.

## Quick Start

### Test on macOS (Sandboxed)

```bash
./tests/test-macos.sh
```

This creates a temporary home directory and runs tests without affecting your actual configuration.

### Test on Linux Distributions (Docker)

Test on a specific distribution:
```bash
./tests/test-linux.sh ubuntu
./tests/test-linux.sh debian
./tests/test-linux.sh fedora
./tests/test-linux.sh arch
```

Test on all distributions:
```bash
./tests/test-linux.sh all
```

## Requirements

### macOS Testing
- macOS (obviously)
- bash
- The script will create a temporary test environment

### Linux Testing
- Docker or Podman installed and running
  - **macOS**: Podman will be auto-installed via Homebrew if neither Docker nor Podman is available
  - **Linux**: Podman will be auto-installed via system package manager if neither Docker nor Podman is available
- Sufficient disk space for container images
- Network connection to download base images
- sudo access (if podman needs to be installed on Linux)
- Homebrew (if running on macOS and podman needs to be installed)

## Test Coverage

### Basic Tests
- ✅ Plan mode (`devboost plan`)
- ✅ Apply mode with dry-run (`devboost apply --dry-run`)
- ✅ Doctor mode (`devboost doctor`)
- ✅ Script syntax validation
- ✅ Config file parsing

### Platform-Specific Tests
- ✅ Ubuntu/Debian (apt package manager)
- ✅ Fedora (dnf package manager)
- ⚠️ Arch Linux (pacman package manager) - Skipped on ARM64 systems (official image limitation)
- ✅ macOS (Homebrew package manager)

## Docker Images

The Docker images are built from:
- `Dockerfile.ubuntu` - Ubuntu 22.04
- `Dockerfile.debian` - Debian Bookworm
- `Dockerfile.fedora` - Fedora Latest
- `Dockerfile.arch` - Arch Linux Latest

Each image includes:
- Basic dependencies (curl, git, sudo, bash)
- A test user with sudo privileges
- The built devboost.sh script

## Manual Testing

### Interactive Container Testing

You can also run interactive tests using docker-compose (requires Docker):

```bash
# Build and run a specific distribution
cd tests/docker
docker-compose build ubuntu
docker-compose run --rm ubuntu

# Inside the container:
/tmp/devboost.sh plan
/tmp/devboost.sh apply --dry-run
/tmp/devboost.sh doctor
```

**Note**: docker-compose requires Docker. For Podman, use the test script directly (`./tests/test-linux.sh [distro]`) or use `podman-compose` if available.

**Arch Linux on ARM64:**
- The official Arch Linux Docker image doesn't support ARM64 architecture
- Tests will automatically skip Arch on ARM64 systems (e.g., Apple Silicon Macs)
- Arch tests should be run on x86_64 systems

### macOS Manual Testing

For more thorough macOS testing, you can use a separate test user:

```bash
# Create a test user (requires admin)
sudo dscl . -create /Users/testdevboost
sudo dscl . -create /Users/testdevboost UserShell /bin/bash
sudo dscl . -create /Users/testdevboost RealName "Test User"
sudo dscl . -create /Users/testdevboost UniqueID 1001
sudo dscl . -create /Users/testdevboost PrimaryGroupID 20
sudo dscl . -create /Users/testdevboost NFSHomeDirectory /Users/testdevboost
sudo createhomedir -c -u testdevboost

# Switch to test user
su - testdevboost

# Run tests
cd /path/to/devboost
./devboost.sh apply
```

## Continuous Integration

These test scripts are designed to be run in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Test Linux
  run: ./tests/test-linux.sh all

- name: Test macOS
  run: ./tests/test-macos.sh
```

## Troubleshooting

### Container Runtime Issues

**Image build fails:**
- Ensure Docker/Podman is running: `docker info` or `podman info`
- Check disk space: `docker system df` or `podman system df`
- Try pulling base images manually: `docker pull ubuntu:22.04` or `podman pull ubuntu:22.04`

**Permission denied:**
- For Docker: Ensure your user is in the docker group (Linux)
- For Podman: Usually works without special permissions (rootless)
- On macOS, Docker Desktop should handle permissions

**Podman installation:**
- The script will attempt to install podman automatically if neither docker nor podman is available
- **macOS**: Installs via Homebrew (`brew install podman`) and automatically initializes the podman machine
- **Linux**: Requires sudo access and a supported Linux distribution (Ubuntu/Debian, Fedora/RHEL, Arch)
- After installation, podman machine is automatically started on macOS

### macOS Test Issues

**Temporary directory issues:**
- The script uses `mktemp` which should work on macOS
- If issues occur, check `/tmp` permissions

**Home directory conflicts:**
- The test uses a temporary directory, so it shouldn't conflict
- If you see issues, check that `$TEST_HOME` is actually temporary

## Adding New Tests

To add a new test:

1. Add test logic to `test-linux.sh` or `test-macos.sh`
2. Update this README with the new test
3. Ensure tests are idempotent (can run multiple times)
4. Document any new requirements

