# 跨平台自动发布方案设计

## 概述

为 starman 建立全自动跨平台分发管道，通过 Git tag 触发 GitHub Actions，利用 GoReleaser 完成编译、归档、包管理器和一键安装脚本的生成与推送。

## 分发矩阵

| 平台 | 安装方式 | 自动化机制 |
|------|----------|------------|
| macOS | `brew install morehao/tap/starman` | GoReleaser brews → 推送到 `morehao/homebrew-tap` |
| Linux | `brew install morehao/tap/starman` | 同上（Linuxbrew） |
| Linux | `curl -fsSL https://... | sh` | install.sh 一键安装脚本 |
| Linux | `.deb` / `.rpm` 包 | GoReleaser nfpms |
| Windows | `scoop bucket add morehao ... && scoop install starman` | GoReleaser scoops → 推送到 `morehao/scoop-bucket` |

## 架构

```
git tag vX.Y.Z
       │
       ▼
┌─────────────────────────────────────────────────┐
│  GitHub Actions: .github/workflows/release.yml  │
│                                                 │
│  1. goreleaser release                          │
│     ├── 交叉编译 (6 目标)                        │
│     ├── tar.gz / zip 归档 ──→ GitHub Release     │
│     ├── checksums.txt ────→ GitHub Release      │
│     ├── .deb / .rpm ──────→ GitHub Release      │
│     ├── Homebrew Formula ──→ morehao/homebrew-tap│
│     └── Scoop Manifest ───→ morehao/scoop-bucket │
│                                                 │
│  2. 渲染 install.sh ────────→ GitHub Release     │
└─────────────────────────────────────────────────┘
```

## 文件变更

### `.goreleaser.yml` 修改

在现有构建和归档配置基础上增加以下模块：

**brews** — 指向 `morehao/homebrew-tap`，自动生成 Ruby Formula 并推送到独立仓库。Formula 包含二进制下载 URL 和 sha256 校验值。

**scoops** — 指向 `morehao/scoop-bucket`，自动生成 JSON Manifest 并推送。

**nfpms** — 为 Linux amd64/arm64 生成 `.deb` 和 `.rpm` 包。主页、许可证、描述等元数据从 GoReleaser 全局字段继承。

**archives** — 增加 `binary.gz` 纯二进制归档格式（不含 README/LICENSE 等额外文件），供 install.sh 脚本直接下载使用。

### `.github/workflows/release.yml` 新建

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
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### `scripts/install.sh` 新建

一键安装脚本模板，功能：

1. 检测 OS（darwin/linux）和架构（amd64/arm64）
2. 构造下载 URL：`https://github.com/morehao/starman/releases/download/v{VERSION}/starman_{VERSION}_{OS}_{ARCH}.tar.gz`
3. 下载、校验 sha256、解压到 `/usr/local/bin`（需 sudo）或 `~/.local/bin`
4. 幂等安装（已安装则提示跳过或覆盖）

### `docs/gorelease.md` 修改

更新为各平台的安装说明文档，列出每个平台的安装命令和前置条件。

## 前置条件

| 条件 | 说明 |
|------|------|
| GitHub 仓库 `morehao/homebrew-tap` | 创建空仓库，存放 Homebrew Formula |
| GitHub 仓库 `morehao/scoop-bucket` | 创建空仓库，存放 Scoop Manifest |
| GitHub Token | Actions 默认的 `GITHUB_TOKEN` 满足读写 Release 和推送公式仓库的需求 |

## 发布流程

1. 修改代码，提交 PR 合并到 main
2. 打 tag：`git tag v0.0.13 && git push origin v0.0.13`
3. GitHub Actions 自动触发，执行 goreleaser：
   - 交叉编译 6 个目标平台
   - 生成 tar.gz/zip/deb/rpm + checksums
   - 创建 GitHub Release 并上传所有产物
   - 推送 Homebrew Formula 到 `morehao/homebrew-tap`
   - 推送 Scoop Manifest 到 `morehao/scoop-bucket`
4. 用户在各平台执行对应的安装命令即可安装最新版本
