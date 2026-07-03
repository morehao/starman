# Cross-Platform Release Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 starman 建立全自动跨平台分发管道——推送 Git tag 触发 GitHub Actions，通过 GoReleaser 自动编译、归档、生成 Homebrew Formula / Scoop Manifest / deb+rpm 包，并推送到对应仓库。

**Architecture:** 修改现有 `.goreleaser.yml` 增加 brews/scoops/nfpms 模块，新建 GitHub Actions 工作流 `.github/workflows/release.yml` 在 tag 推送时触发 goreleaser，新建 `scripts/install.sh` 一键安装脚本，更新 `docs/gorelease.md` 为安装说明文档。

**Tech Stack:** GoReleaser v2, GitHub Actions, Homebrew Ruby Formula, Scoop JSON Manifest, nfpms, Shell

## Global Constraints

- GoReleaser v2 配置格式（项目已使用）
- CGO_ENABLED=0，纯静态编译
- 交叉编译目标：linux/darwin/windows × amd64/arm64
- Homebrew Formula 推送到 `morehao/homebrew-tap`
- Scoop Manifest 推送到 `morehao/scoop-bucket`
- 发布触发：Git tag（`v*` 模式）
- GitHub Actions 使用默认 `GITHUB_TOKEN`
- 项目描述：CLI tool for managing GitHub starred repositories with AI — sync, analyze, categorize, search, and generate awesome lists

---

### Task 1: 更新 `.goreleaser.yml` — 增加 brews/scoops/nfpms 模块

**Files:**
- Modify: `.goreleaser.yml`

**Interfaces:**
- Consumes: 现有的 builds / archives / checksum / changelog 配置
- Produces: 扩展后的 `.goreleaser.yml`，包含 brews、scoops、nfpms 模块

- [ ] **Step 1: 修改 `.goreleaser.yml`**

在现有文件末尾追加以下配置：

```yaml
archives:
  - id: default
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
  - id: binary
    format: binary

nfpms:
  - id: starman
    package_name: starman
    file_name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    vendor: morehao
    homepage: https://github.com/morehao/starman
    maintainer: morehao <morehao@users.noreply.github.com>
    description: >-
      CLI tool for managing GitHub starred repositories with AI —
      sync, analyze, categorize, search, and generate awesome lists
    license: Apache-2.0
    formats:
      - deb
      - rpm
    contents:
      - src: ./scripts/install.sh
        dst: /usr/local/bin/install-starman.sh
    scripts:
      postinstall: ./scripts/postinstall.sh

brews:
  - name: starman
    repository:
      owner: morehao
      name: homebrew-tap
    homepage: https://github.com/morehao/starman
    description: >-
      CLI tool for managing GitHub starred repositories with AI —
      sync, analyze, categorize, search, and generate awesome lists
    license: Apache-2.0
    test: |
      system "#{bin}/starman --version"
    install: |
      bin.install "starman"

scoops:
  - name: starman
    repository:
      owner: morehao
      name: scoop-bucket
    homepage: https://github.com/morehao/starman
    description: >-
      CLI tool for managing GitHub starred repositories with AI —
      sync, analyze, categorize, search, and generate awesome lists
    license: Apache-2.0
```

同时将原来的 `archives:` 块改为带 id 的多 archives 格式：

```yaml
archives:
  - id: default
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
  - id: binary
    format: binary
```

- [ ] **Step 2: 验证 GoReleaser 配置语法**

运行：`goreleaser check --config .goreleaser.yml`
预期：输出 `config is valid` 且无报错

- [ ] **Step 3: 提交**

```bash
git add .goreleaser.yml
git commit -m "build: add brews/scoops/nfpms to goreleaser config"
```

---

### Task 2: 新建 GitHub Actions 发布工作流

**Files:**
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: `.goreleaser.yml`（Task 1 产物）
- Produces: 无（被 GitHub Actions 调用）、后续 GitHub Release 产物

- [ ] **Step 1: 创建 `.github/workflows/release.yml`**

```yaml
name: Release

on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: 提交**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add GitHub Actions release workflow"
```

---

### Task 3: 新建 `scripts/install.sh` 一键安装脚本

**Files:**
- Create: `scripts/install.sh`

**Interfaces:**
- Consumes: 无（独立脚本，从 GitHub Releases API 获取最新版本）
- Produces: 可被用户直接 `curl | sh` 执行的安装脚本

- [ ] **Step 1: 创建 `scripts/install.sh`**

```sh
#!/bin/sh
set -e

REPO="morehao/starman"
BIN="starman"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

info()  { printf "${GREEN}%s${NC}\n" "$*"; }
error() { printf "${RED}%s${NC}\n" "$*" >&2; exit 1; }

detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$OS" in
    linux|darwin) ;;
    *) error "Unsupported OS: $OS" ;;
  esac

  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) error "Unsupported architecture: $ARCH" ;;
  esac
}

get_latest_version() {
  VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

  if [ -z "$VERSION" ]; then
    error "Failed to get latest version"
  fi
}

install() {
  TARBALL="${BIN}_${VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

  TMPDIR=$(mktemp -d)
  trap 'rm -rf "$TMPDIR"' EXIT

  info "Downloading ${TARBALL}..."
  curl -sL "$URL" -o "$TMPDIR/$TARBALL" || error "Download failed"

  tar xzf "$TMPDIR/$TARBALL" -C "$TMPDIR"

  INSTALL_DIR="/usr/local/bin"
  if [ ! -w "$INSTALL_DIR" ]; then
    info "Installing to ${HOME}/.local/bin (no write permission on ${INSTALL_DIR})"
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi

  mv "$TMPDIR/$BIN" "$INSTALL_DIR/$BIN"
  chmod +x "$INSTALL_DIR/$BIN"

  info "starman ${VERSION} installed to ${INSTALL_DIR}/${BIN}"

  if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    printf "Add %s to your PATH:\n  export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR" "$INSTALL_DIR"
  fi
}

detect_platform
get_latest_version
install
```

- [ ] **Step 2: 赋可执行权限并验证语法**

```bash
chmod +x scripts/install.sh && sh -n scripts/install.sh
```

- [ ] **Step 3: 提交**

```bash
git add scripts/install.sh
git commit -m "feat: add one-line install script"
```

---

### Task 4: 更新 `docs/gorelease.md` 为安装说明文档

**Files:**
- Modify: `docs/gorelease.md`

**Interfaces:**
- Consumes: 无
- Produces: 面向用户的跨平台安装指南

- [ ] **Step 1: 重写 `docs/gorelease.md`**

```markdown
# 安装 starman

## macOS

```sh
brew install morehao/tap/starman
```

## Linux

### Homebrew (Linuxbrew)

```sh
brew install morehao/tap/starman
```

### 一键安装脚本

```sh
curl -fsSL https://raw.githubusercontent.com/morehao/starman/main/scripts/install.sh | sh
```

### deb 包 (Debian/Ubuntu)

从 [GitHub Releases](https://github.com/morehao/starman/releases/latest) 下载 `.deb` 文件：

```sh
dpkg -i starman_*.deb
```

### rpm 包 (Fedora/RHEL)

从 [GitHub Releases](https://github.com/morehao/starman/releases/latest) 下载 `.rpm` 文件：

```sh
rpm -i starman_*.rpm
```

## Windows

```powershell
scoop bucket add morehao https://github.com/morehao/scoop-bucket
scoop install starman
```

## 手动下载

所有平台的二进制包可在 [GitHub Releases](https://github.com/morehao/starman/releases/latest) 页面下载。
```

- [ ] **Step 2: 提交**

```bash
git add docs/gorelease.md
git commit -m "docs: update installation guide with cross-platform instructions"
```

---

### Task 5: 确保 `GITHUB_TOKEN` 权限满足推送外部仓库

**说明：** GitHub Actions 默认的 `GITHUB_TOKEN` 只有当前仓库的写入权限。GoReleaser 需要向 `morehao/homebrew-tap` 和 `morehao/scoop-bucket` 推送提交，默认 token 无法做到。

**解决方案：** 使用 Personal Access Token (PAT) 替代默认 token。

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: 修改 release.yml 中的 token**

将：

```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

替换为：

```yaml
env:
  GITHUB_TOKEN: ${{ secrets.RELEASE_TOKEN }}
```

同时增加 Scoop bucket 的 commit 作者配置（在 scoops 块中），避免 scoop-bucket 提交使用默认的 actions 用户。

- [ ] **Step 2: 同步更新 `.goreleaser.yml` 中 scoops 配置，增加 commit_author**

在 scoops 块中增加：

```yaml
scoops:
  - name: starman
    repository:
      owner: morehao
      name: scoop-bucket
    commit_author:
      name: morehao
      email: morehao@users.noreply.github.com
```

- [ ] **Step 3: 提交**

```bash
git add .github/workflows/release.yml .goreleaser.yml
git commit -m "ci: use PAT for cross-repo push access"
```

**用户操作：** 需要在 GitHub 仓库 Settings > Secrets and variables > Actions 中创建 `RELEASE_TOKEN` secret，值为一个具有 `repo` scope 的 Personal Access Token (classic) 或 Fine-grained PAT（对 `morehao/starman`、`morehao/homebrew-tap`、`morehao/scoop-bucket` 三个仓库有 Contents write 权限）。

---

### Task 6: 创建 `scripts/postinstall.sh`（nfpms postinstall 脚本）

**Files:**
- Create: `scripts/postinstall.sh`

**Interfaces:**
- Consumes: nfpms `postinstall` 配置（Task 1）
- Produces: deb/rpm 安装后自动执行的脚本

- [ ] **Step 1: 创建 `scripts/postinstall.sh`**

```sh
#!/bin/sh
echo "starman installed successfully. Run 'starman --help' to get started."
```

- [ ] **Step 2: 赋可执行权限**

```bash
chmod +x scripts/postinstall.sh
```

- [ ] **Step 3: 提交**

```bash
git add scripts/postinstall.sh
git commit -m "feat: add postinstall script for nfpms"
```

---

### Task 7: 移除 nfpms 的 install.sh 和 postinstall.sh 引用（如果不需要）

**说明：** Task 1 中 nfpms 的 `contents` 和 `scripts.postinstall` 引用了 install.sh 和 postinstall.sh，但这些文件是打包进 deb/rpm 的。install.sh 作为 deb/rpm 内容不合理且 postinstall.sh 意义不大，移除这些配置保持简洁。

**Files:**
- Modify: `.goreleaser.yml`

- [ ] **Step 1: 移除 nfpms 的 contents 和 scripts 块**

删除 nfpms 块中的以下内容：

```yaml
    contents:
      - src: ./scripts/install.sh
        dst: /usr/local/bin/install-starman.sh
    scripts:
      postinstall: ./scripts/postinstall.sh
```

使 nfpms 块变为：

```yaml
nfpms:
  - id: starman
    package_name: starman
    file_name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    vendor: morehao
    homepage: https://github.com/morehao/starman
    maintainer: morehao <morehao@users.noreply.github.com>
    description: >-
      CLI tool for managing GitHub starred repositories with AI —
      sync, analyze, categorize, search, and generate awesome lists
    license: Apache-2.0
    formats:
      - deb
      - rpm
```

- [ ] **Step 2: 移除 Task 6 的 postinstall.sh**

```bash
git rm scripts/postinstall.sh
```

- [ ] **Step 3: 提交**

```bash
git add .goreleaser.yml scripts/postinstall.sh
git commit -m "build: remove unnecessary nfpms contents and postinstall"
```

---

### Task 8: 最终整理 — 合并优化

**说明：** 以上 Task 按逻辑拆分，为保证每个 Task 独立可测，实际执行时可以将 Task 1-7 合并为更高效的提交序列。

**推荐合并提交顺序：**
1. 修改 `.goreleaser.yml`（brews + scoops + nfpms，含 commit_author）
2. 新建 `.github/workflows/release.yml`（使用 `RELEASE_TOKEN`）
3. 新建 `scripts/install.sh`
4. 更新 `docs/gorelease.md`

共 4 个提交，避免 Task 6/7 的往返修改。

---

### Task 9: 将 install.sh 上传到 GitHub Release 产物

**Files:**
- Modify: `.goreleaser.yml`

**Interfaces:**
- Consumes: `scripts/install.sh`
- Produces: install.sh 作为 Release 产物上传（兼容 `curl | sh` 场景外的手动下载）

- [ ] **Step 1: 在 `.goreleaser.yml` 中添加 extra_files**

在 archives 的 `id: default` 块中增加：

```yaml
    files:
      - src: scripts/install.sh
        dst: .
```

完整 archives 配置变为：

```yaml
archives:
  - id: default
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    files:
      - src: scripts/install.sh
        dst: .
  - id: binary
    format: binary
```

- [ ] **Step 2: 提交**

```bash
git add .goreleaser.yml
git commit -m "build: include install.sh in release archives"
```
