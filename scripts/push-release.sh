#!/usr/bin/env bash
# 推送规则: 推送当前分支 → 读取最新版本 → 增量 bump → 建 tag + 推送 → 创建 GitHub Release。
# 用法:
#   ./scripts/push-release.sh                 # bump PATCH (v1.2.3 -> v1.2.4)
#   ./scripts/push-release.sh --minor         # bump MINOR
#   ./scripts/push-release.sh --major         # bump MAJOR
#   ./scripts/push-release.sh --version v2.0.0
# 前置: git + gh CLI 已登录(gh auth status);无 tag 时从 v0.0.0 起步。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# --- 1. 参数: bump 类型或显式版本 ---
BUMP="patch"
VERSION_OVERRIDE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --patch)  BUMP="patch" ;;
    --minor)  BUMP="minor" ;;
    --major)  BUMP="major" ;;
    --version) VERSION_OVERRIDE="$2"; shift ;;
    *) echo "未知参数: $1" >&2; exit 2 ;;
  esac
  shift
done

# --- 2. 读取最新版本(全仓库 tag,取最高 semver;无则从 v0.0.0 起步) ---
latest() {
  git tag -l 'v[0-9]*' | sort -V | tail -n1
}
LATEST="$(latest || true)"
LATEST="${LATEST:-v0.0.0}"

bump() {  # bump <version> <major|minor|patch>
  local ver="$1" part="$2"
  [[ "$ver" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] || { echo "非法版本: $ver" >&2; exit 2; }
  local maj="${BASH_REMATCH[1]}" min="${BASH_REMATCH[2]}" pat="${BASH_REMATCH[3]}"
  case "$part" in
    major) maj=$((maj+1)); min=0; pat=0 ;;
    minor) min=$((min+1)); pat=0 ;;
    patch) pat=$((pat+1)) ;;
  esac
  echo "v${maj}.${min}.${pat}"
}

if [[ -n "$VERSION_OVERRIDE" ]]; then
  NEW="$VERSION_OVERRIDE"
  [[ "$NEW" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "版本须形如 vX.Y.Z: $NEW" >&2; exit 2; }
else
  NEW="$(bump "$LATEST" "$BUMP")"
fi

# --- 3. 护栏 ---
if ! git rev-parse --git-dir >/dev/null 2>&1; then echo "非 git 仓库" >&2; exit 1; fi
if ! git diff --quiet; then echo "工作区有未提交改动,先提交再发布" >&2; exit 1; fi
if git rev-parse "$NEW" >/dev/null 2>&1; then echo "tag 已存在: $NEW(如需重发先删除)" >&2; exit 1; fi

# --- 4. 推送当前分支代码 ---
BRANCH="$(git branch --show-current)"
echo "==> push 分支 ${BRANCH} (基于版本 ${LATEST})"
git push origin "$BRANCH"

# --- 5. 增量建 tag + 推送 ---
echo "==> 创建并推送 tag $NEW"
git tag -a "$NEW" -m "Release $NEW"
git push origin "$NEW"

# --- 6. 创建 GitHub Release(generate-notes 自动收录自上一 tag 以来的提交) ---
if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
  echo "==> gh release create $NEW"
  gh release create "$NEW" --generate-notes
else
  echo "==> 跳过 Release(未安装 gh 或未登录);tag 已推送,可手动:"
  echo "    gh release create $NEW --generate-notes"
fi

echo "==> 完成: $LATEST -> $NEW"