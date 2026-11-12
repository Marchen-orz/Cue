#!/usr/bin/env bash
# 构建 Cue（Git Bash / WSL）：
#   结束占用进程 → wails 构建 → 清除 build 目录的 Low 完整性标签 → 校验
#
# 为什么要清标签：Codex 沙箱服务会把项目目录打上 Low 完整性标签，构建出的
# build/bin/Cue.exe 继承后以低权限运行，写不了 %APPDATA%\Cue（保存设置报权限不足）。
set -euo pipefail
cd "$(dirname "$0")"

# 1. 结束正在运行的 Cue（占用会导致覆盖失败）
taskkill //F //IM Cue.exe >/dev/null 2>&1 || true

# 2. 定位 wails
GOPATH_BIN="$(go env GOPATH)/bin"
WAILS="$GOPATH_BIN/wails.exe"
if [ ! -x "$WAILS" ]; then WAILS="wails"; fi

# 3. 构建（正式发布参数，与 README 一致）
echo "==> wails build (prod)"
"$WAILS" build -ldflags "-s -w" -tags prod

# 4. 清除 build 目录及其下文件的完整性标签（幂等）
echo "==> 清除 build 目录完整性标签"
go run ./tools/clrlabel clear build

# 5. 校验：应输出 none
echo "==> 校验完整性标签"
go run ./tools/clrlabel query build/bin/Cue.exe

echo "==> 完成: build/bin/Cue.exe"
