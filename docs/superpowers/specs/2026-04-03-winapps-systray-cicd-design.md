# WinApps Systray CI/CD Pipeline Design

## Overview
This design outlines the Continuous Integration and Continuous Deployment (CI/CD) strategy for the `winapps_systray` project. The pipeline automates testing, linting, and building of `.deb` and `.rpm` packages for both `amd64` (x86) and `arm64` architectures, culminating in automated GitHub Releases.

## Architecture & Triggers
The CI/CD pipeline runs on GitHub Actions.
- **Pull Requests & Main Branch:** Every push to `main` and all pull requests trigger the testing and linting jobs to ensure code quality and prevent regressions.
- **Tags:** Pushing a semantic version tag (e.g., `v1.0.0`) triggers the full release pipeline, producing artifacts and creating a GitHub Release.
- **Multi-Arch Strategy:** Due to the reliance on CGo (`gotk3`), cross-compilation is avoided in favor of GitHub's native runners. The build matrix uses `ubuntu-latest` (amd64) and `ubuntu-24.04-arm` (arm64).

## Components & Data Flow

### 1. Linting & Quality
- **Job:** `lint`
- **Action:** `golangci/golangci-lint-action`
- **Purpose:** Enforces Go code standards. Runs on `ubuntu-latest`.

### 2. Testing
- **Job:** `test`
- **Matrix:** `ubuntu-latest`, `ubuntu-24.04-arm`
- **Dependencies:** `libgtk-3-dev`, `libayatana-appindicator3-dev` installed via `apt-get`.
- **Action:** Runs `go test -v ./...` natively to ensure code correctness on both architectures.

### 3. Packaging & nfpm Modifications
- **Tool:** `goreleaser/nfpm-action` or a shell script invoking `make deb` and `make rpm`.
- **nfpm Config Update:** Modify `nfpm.yaml` to dynamically set the architecture using environment variables (e.g., `arch: ${GOARCH}`).
- **Data Flow:** The artifacts (`.deb` and `.rpm`) are temporarily stored using `actions/upload-artifact`.

### 4. Release Publishing
- **Job:** `release`
- **Condition:** Runs only on tag pushes (`if: startsWith(github.ref, 'refs/tags/v')`) and after `test` and `build` jobs succeed.
- **Action:** Downloads all uploaded artifacts and creates a unified GitHub Release using `softprops/action-gh-release` with the compiled packages.

## Additional DevOps
- **Dependabot:** Automate updates for Go modules and GitHub Actions using `.github/dependabot.yml`.

## Open Questions & Risks
- **Testing Constraints:** While `ubuntu-24.04-arm` is free for open source, if the repo becomes private, it incurs costs.
- **GTK Headless Testing:** `go test` with `gotk3` may require a virtual display (`xvfb`) if UI tests are added in the future.
