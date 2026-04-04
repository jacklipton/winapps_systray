# WinApps Systray CI/CD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a GitHub Actions CI/CD pipeline for testing, linting, and building x86/arm64 packages.

**Architecture:** GitHub Actions matrix utilizing native `ubuntu-latest` and `ubuntu-24.04-arm` runners.

**Tech Stack:** GitHub Actions, Dependabot, nFPM, Makefile, golangci-lint.

---

### Task 1: Make nFPM Architecture Dynamic

**Files:**
- Modify: `nfpm.yaml`

- [ ] **Step 1: Replace hardcoded `amd64` architecture**

In `nfpm.yaml`, replace `arch: amd64` with `arch: ${GOARCH}`.

```yaml
name: winapps-systray
arch: ${GOARCH}
version: 0.1.0
```

- [ ] **Step 2: Commit**

```bash
git add nfpm.yaml
git commit -m "build: make nfpm architecture dynamic via GOARCH"
```

### Task 2: Configure Dependabot

**Files:**
- Create: `.github/dependabot.yml`

- [ ] **Step 1: Write Dependabot configuration**

Create `.github/dependabot.yml` with the following content:

```yaml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "monthly"
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "monthly"
```

- [ ] **Step 2: Commit**

```bash
git add .github/dependabot.yml
git commit -m "ci: add dependabot configuration"
```

### Task 3: Create CI/CD Workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Write the GitHub Actions workflow**

Create `.github/workflows/ci.yml` with the following content:

```yaml
name: CI/CD

on:
  push:
    branches: [ "main" ]
    tags: [ "v*.*.*" ]
  pull_request:
    branches: [ "main" ]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - name: Install Dependencies (GTK)
        run: |
          sudo apt-get update
          sudo apt-get install -y libgtk-3-dev libayatana-appindicator3-dev
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m

  build-and-test:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            arch: amd64
          - os: ubuntu-24.04-arm
            arch: arm64
    runs-on: ${{ matrix.os }}
    needs: lint
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          
      - name: Install System Dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y libgtk-3-dev libayatana-appindicator3-dev
          
      - name: Run Tests
        env:
          CGO_ENABLED: "1"
        run: go test -v ./...
        
      - name: Install nfpm
        run: |
          go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
          
      - name: Build Packages (deb & rpm)
        env:
          CGO_ENABLED: "1"
          GOARCH: ${{ matrix.arch }}
        run: |
          make deb
          make rpm
          
      - name: Upload Artifacts
        uses: actions/upload-artifact@v4
        with:
          name: packages-${{ matrix.arch }}
          path: |
            build/*.deb
            build/*.rpm

  release:
    needs: build-and-test
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/v')
    steps:
      - uses: actions/download-artifact@v4
        with:
          path: packages
          pattern: packages-*
          merge-multiple: true
          
      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            packages/*.deb
            packages/*.rpm
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add multi-arch github actions workflow for testing and release"
```
