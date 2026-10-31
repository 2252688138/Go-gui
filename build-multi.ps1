<#
.SYNOPSIS
GoGUI 多端构建脚本：一套 UI 代码，一次产出 Windows / Linux / macOS / Web 四端产物。

.DESCRIPTION
与 build.ps1 的区别：
  build.ps1        只管 Windows 发布（图标资源 + NSIS 安装器 + UPX）
  build-multi.ps1  跑「多端编译矩阵」，验证并产出其它平台的二进制

版本号与 build.ps1 共用 VERSION.txt，通过 -ldflags -X 注入 gogui/internal/version.*；
桌面端一律 CGO_ENABLED=0（纯 Go + purego 动态加载系统库），产物无外部依赖。

.EXAMPLE
./build-multi.ps1                 # 全矩阵构建到 dist/
./build-multi.ps1 -Only linux     # 只构建 Linux（amd64 + arm64）
./build-multi.ps1 -Only web -Serve  # 构建 wasm 并起本地静态服务器
#>
param(
    [string]$OutDir = 'dist',
    [ValidateSet('all', 'windows', 'linux', 'darwin', 'web')]
    [string]$Only = 'all',
    # 仅 web 目标：构建完启动本地静态服务器并打开浏览器
    [switch]$Serve,
    [int]$Port = 8089,
    # 附加 UPX 压缩（仅 windows/linux amd64 生效）
    [switch]$Upx
)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)

# ================= 版本信息 =================
$versionFile = Join-Path $PSScriptRoot 'VERSION.txt'
$Version = '0.0.0'
if (Test-Path -LiteralPath $versionFile) {
    $Version = (Get-Content -LiteralPath $versionFile -TotalCount 1).Trim()
}
if (-not $Version) { $Version = '0.0.0' }

$Commit = 'none'
try { $Commit = (git rev-parse --short HEAD 2>$null) } catch { }
if (-not $Commit) { $Commit = 'none' }
$BuildTime = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

$ldflags = @(
    '-s', '-w', '-buildid=',
    "-X gogui/internal/version.Number=$Version",
    "-X gogui/internal/version.Commit=$Commit",
    "-X gogui/internal/version.BuildTime=$BuildTime",
    '-X gogui/internal/version.Channel=dev'
) -join ' '

if (-not (Test-Path -LiteralPath $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir | Out-Null
}
$OutDirAbs = (Resolve-Path $OutDir).Path

# ================= 构建矩阵 =================
# 每项：名称 / GOOS / GOARCH / 是否加 windowsgui / 输出文件
$matrix = @(
    @{ Name = 'windows-amd64'; GOOS = 'windows'; GOARCH = 'amd64'; GUI = $true;  Out = 'gogui-windows-amd64.exe'; Pkg = './cmd/gallery' },
    @{ Name = 'linux-amd64';   GOOS = 'linux';   GOARCH = 'amd64'; GUI = $false; Out = 'gogui-linux-amd64';       Pkg = './cmd/gallery' },
    @{ Name = 'linux-arm64';   GOOS = 'linux';   GOARCH = 'arm64'; GUI = $false; Out = 'gogui-linux-arm64';       Pkg = './cmd/gallery' },
    @{ Name = 'darwin-amd64';  GOOS = 'darwin';  GOARCH = 'amd64'; GUI = $false; Out = 'gogui-darwin-amd64';      Pkg = './cmd/gallery' },
    @{ Name = 'darwin-arm64';  GOOS = 'darwin';  GOARCH = 'arm64'; GUI = $false; Out = 'gogui-darwin-arm64';      Pkg = './cmd/gallery' },
    @{ Name = 'wasm';          GOOS = 'js';      GOARCH = 'wasm';  GUI = $false; Out = 'web\main.wasm';          Pkg = './cmd/gallery' }
)

if ($Only -ne 'all') {
    # web 是 wasm 目标的别名（命令行习惯写 -Only web）
    $matrix = $matrix | Where-Object { $_.Name -like "$Only*" -or ($Only -eq 'web' -and $_.GOOS -eq 'js') }
}
if ($matrix.Count -eq 0) {
    Write-Host ("[warn] 没有匹配的目标：{0}" -f $Only) -ForegroundColor Yellow
    exit 1
}

$results = @()
$failed = @()

foreach ($m in $matrix) {
    Write-Host ("[build] {0,-14} {1}/{2}" -f $m.Name, $m.GOOS, $m.GOARCH) -ForegroundColor Cyan

    $env:CGO_ENABLED = '0'
    $env:GOOS = $m.GOOS
    $env:GOARCH = $m.GOARCH
    $env:GOAMD64 = if ($m.GOARCH -eq 'amd64') { 'v1' } else { $null }

    $out = Join-Path $OutDirAbs $m.Out
    $flags = $ldflags
    if ($m.GUI) { $flags = "$flags -H windowsgui" }

    & go build -trimpath -buildvcs=false -ldflags $flags -o $out $m.Pkg
    if ($LASTEXITCODE -ne 0) {
        Write-Host ("[fail ] {0}" -f $m.Name) -ForegroundColor Red
        $failed += $m.Name
        continue
    }

    # wasm 需要配套运行时：wasm_exec.js + 宿主页 + 字体，才能直接预览
    if ($m.GOOS -eq 'js') {
        $webDir = Split-Path -Parent $out
        $goroot = (& go env GOROOT).Trim()
        $execJs = Join-Path $goroot 'lib\wasm\wasm_exec.js'
        if (-not (Test-Path -LiteralPath $execJs)) { $execJs = Join-Path $goroot 'misc\wasm\wasm_exec.js' }
        if (Test-Path -LiteralPath $execJs) {
            Copy-Item -LiteralPath $execJs -Destination (Join-Path $webDir 'wasm_exec.js') -Force
        }
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'web\app.html') -Destination (Join-Path $webDir 'index.html') -Force
        # hls.js：Web 端播放 .m3u8（Chrome 系不原生支持 HLS）
        $hlsJs = Join-Path $PSScriptRoot 'web\hls.min.js'
        if (Test-Path -LiteralPath $hlsJs) {
            Copy-Item -LiteralPath $hlsJs -Destination (Join-Path $webDir 'hls.min.js') -Force
        }
        # 字体：中文字形必须由页面提供（wasm 侧不能读系统字体）
        foreach ($f in @('C:\Windows\Fonts\simhei.ttf', 'C:\Windows\Fonts\msyh.ttc')) {
            if (Test-Path -LiteralPath $f) {
                Copy-Item -LiteralPath $f -Destination (Join-Path $webDir 'font.ttf') -Force
                break
            }
        }
    }

    if ($Upx -and $m.GOARCH -eq 'amd64' -and $m.GOOS -ne 'js') {
        $upx = Get-Command upx -ErrorAction SilentlyContinue
        if ($upx) {
            & $upx.Source -9 -q $out | Out-Null
        }
    }

    $results += (Get-Item -LiteralPath $out)
}

# ================= 汇总 =================
Write-Host ''
'{0,-26} {1,10}' -f '产物', '体积(MB)' | Write-Host
'-' * 38 | Write-Host
foreach ($f in $results) {
    $size = $f.Length / 1MB
    '{0,-26} {1,10:N2}' -f $f.Name, $size | Write-Host
}
Write-Host ''
Write-Host ("版本 {0} · 提交 {1} · 目标 {2}" -f $Version, $Commit, $Only) -ForegroundColor Green

if ($failed.Count -gt 0) {
    Write-Host ("构建失败：{0}" -f ($failed -join ', ')) -ForegroundColor Red
    exit 1
}

# web：起一个静态服务器方便直接打开（与 ./build.ps1 -Target web -Serve 等价）
if ($Serve -and (Test-Path (Join-Path $OutDirAbs 'web\main.wasm'))) {
    $url = "http://127.0.0.1:$Port/"
    Write-Host ("[serve] {0}" -f $url) -ForegroundColor Cyan
    # 必须回到宿主平台：GOOS=js 的 go run 无法在本机执行
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    & go run ./web/server -dir (Join-Path $OutDirAbs 'web') -addr ("127.0.0.1:{0}" -f $Port) -open
}
