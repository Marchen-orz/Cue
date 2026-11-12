# 构建 Cue（PowerShell）：
#   结束占用进程 -> wails 构建 -> 清除 build 目录的 Low 完整性标签 -> 校验
#
# 为什么要清标签：Codex 沙箱服务会把项目目录打上 Low 完整性标签，构建出的
# build\bin\Cue.exe 继承后以低权限运行，写不了 %APPDATA%\Cue（保存设置报权限不足）。

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)

# 1. 结束正在运行的 Cue
Get-Process Cue -ErrorAction SilentlyContinue | Stop-Process -Force

# 2. 定位 wails
$wails = Join-Path (go env GOPATH) 'bin/wails.exe'
if (-not (Test-Path $wails)) { $wails = 'wails' }

# 3. 构建（正式发布参数，与 README 一致）
Write-Host '==> wails build (prod)'
& $wails build -ldflags "-s -w" -tags prod
if ($LASTEXITCODE -ne 0) { throw 'wails build 失败' }

# 4. 清除 build 目录及其下文件的完整性标签（幂等）
Write-Host '==> 清除 build 目录完整性标签'
go run ./tools/clrlabel clear build
if ($LASTEXITCODE -ne 0) { throw '清除完整性标签失败' }

# 5. 校验：应输出 none
Write-Host '==> 校验完整性标签'
go run ./tools/clrlabel query build/bin/Cue.exe

Write-Host '==> 完成: build\bin\Cue.exe'
