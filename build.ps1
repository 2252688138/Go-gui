<#
.SYNOPSIS
GoGUI 优化构建脚本（体积优先）。

.DESCRIPTION
默认发布构建参数：
  -trimpath -buildvcs=false                去掉本机绝对路径与 VCS 信息（可复现、体积更小）
  -ldflags "-s -w -H windowsgui -buildid=" 去掉符号表/调试信息/构建 ID，GUI 子系统不弹控制台
  CGO_ENABLED=0                            纯 syscall 实现，无 cgo 依赖，产物更小且免运行库
  GOAMD64=v1                               保守指令集，兼容旧 CPU（自用机器可改 v3 换性能）

实测（gallery，go1.26）：
  默认 go build            6.45 MB
  本脚本发布构建           4.54 MB   (-30%)
  本脚本发布构建 + -Upx    1.52 MB   (-76%)

-Upx 说明：启动时整份镜像自解压进私有内存（常驻内存略增），压缩后的 Go 二进制偶发被
杀软误报；不追求极限体积时建议不带该开关。

.EXAMPLE
./build.ps1                     # 发布构建 gallery -> dist\gallery.exe
./build.ps1 -Target all         # gallery + demo + 三个调试工具
./build.ps1 -Upx                # 再叠加 UPX -9
./build.ps1 -Dev                # 开发构建：保留符号、带控制台（stderr 日志可见）
#>
param(
    [ValidateSet('gallery', 'demo', 'tools', 'all', 'web')]
    [string]$Target = 'gallery',
    [switch]$Upx,
    [switch]$Dev,
    [string]$OutDir = 'dist',
    # 仅 web 目标：构建完启动本地静态服务器并打开浏览器
    [switch]$Serve,
    [int]$Port = 8088
)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)

# ---- Web 目标：同一套 ui 代码编译成 WebAssembly（canvas + WebGL 2）----
if ($Target -eq 'web') {
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'js'
    $env:GOARCH = 'wasm'
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

    Write-Host '[web 1/4] go build ./cmd/gallery -> main.wasm' -ForegroundColor Cyan
    & go build -trimpath -buildvcs=false -ldflags '-s -w' -o (Join-Path $OutDir 'main.wasm') ./cmd/gallery
    if ($LASTEXITCODE -ne 0) { throw 'go build 失败：./cmd/gallery' }

    Write-Host '[web 2/4] 复制 wasm_exec.js' -ForegroundColor Cyan
    $goroot = (& go env GOROOT).Trim()
    $exec = Join-Path $goroot 'lib\wasm\wasm_exec.js'
    if (-not (Test-Path $exec)) { $exec = Join-Path $goroot 'misc\wasm\wasm_exec.js' }
    Copy-Item $exec (Join-Path $OutDir 'wasm_exec.js') -Force

    Write-Host '[web 3/4] 复制 index.html' -ForegroundColor Cyan
    Copy-Item (Join-Path $PSScriptRoot 'web\app.html') (Join-Path $OutDir 'index.html') -Force

    Write-Host '[web 4/4] 复制字体 -> font.ttf' -ForegroundColor Cyan
    $font = 'C:\Windows\Fonts\simhei.ttf'
    if (-not (Test-Path $font)) { $font = 'C:\Windows\Fonts\msyh.ttc' }
    if (-not (Test-Path $font)) { throw "找不到可用字体：$font" }
    Copy-Item $font (Join-Path $OutDir 'font.ttf') -Force

    $w = Get-Item (Join-Path $OutDir 'main.wasm')
    Write-Host ''
    Write-Host ('main.wasm  {0:N2} MB' -f ($w.Length / 1MB)) -ForegroundColor Green
    Write-Host ('输出目录   {0}' -f (Resolve-Path $OutDir)) -ForegroundColor Green
    Write-Host ('本地预览   http://127.0.0.1:{0}/   (加 -Serve 自动打开)' -f $Port) -ForegroundColor Yellow

    if ($Serve) {
        # 起本地服务器必须回到宿主平台，否则 GOOS=js 的 go run 无法执行
        $env:GOOS = 'windows'
        $env:GOARCH = 'amd64'
        & go run ./web/server -dir $OutDir -addr ('127.0.0.1:{0}' -f $Port) -open
    }
    return
}

$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
if (-not $env:GOAMD64) { $env:GOAMD64 = 'v1' }

if ($Dev) {
    $ldArgs = '-H windowsgui'
    $extra = @('-buildvcs=false')
} else {
    $ldArgs = '-s -w -H windowsgui -buildid='
    $extra = @('-trimpath', '-buildvcs=false')
}

$targets = @{
    gallery = @(@{ Name = 'gallery.exe'; Pkg = './cmd/gallery' })
    demo    = @(@{ Name = 'demo.exe'; Pkg = './cmd/demo' })
    tools   = @(
        @{ Name = 'atlasdump.exe'; Pkg = './tools/atlasdump' },
        @{ Name = 'fonttest.exe'; Pkg = './tools/fonttest' },
        @{ Name = 'layouttest.exe'; Pkg = './tools/layouttest' }
    )
}

$list = switch ($Target) {
    'all' { $targets.gallery + $targets.demo + $targets.tools }
    default { $targets[$Target] }
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$results = @()
foreach ($t in $list) {
    $out = Join-Path $OutDir $t.Name
    Write-Host ("[build] {0} <- {1}" -f $out, $t.Pkg) -ForegroundColor Cyan
    & go build @extra -ldflags $ldArgs -o $out $t.Pkg
    if ($LASTEXITCODE -ne 0) { throw "go build 失败：$($t.Pkg)" }
    $results += Get-Item $out
}

if ($Upx) {
    $candidates = @()
    if ($env:UPX_EXE) { $candidates += $env:UPX_EXE }
    $candidates += @('H:\JIMaX\bin\upx.exe', '.\bin\upx.exe', '.\third_party\upx.exe')
    $upxPath = $null
    foreach ($c in $candidates) {
        if (-not $c) { continue }
        if (Test-Path -LiteralPath $c) { $upxPath = $c; break }
    }
    if (-not $upxPath) {
        $cmd = Get-Command upx -ErrorAction SilentlyContinue
        if ($cmd) { $upxPath = $cmd.Source }
    }
    if (-not $upxPath) {
        Write-Warning '未找到 upx.exe（可用环境变量 UPX_EXE 指定），跳过压缩'
    } else {
        foreach ($f in $results) {
            Write-Host ("[upx] {0}" -f $f.Name) -ForegroundColor Yellow
            & $upxPath -9 -q $f.FullName
            if ($LASTEXITCODE -ne 0) { Write-Warning "UPX 压缩失败：$($f.Name)" }
        }
    }
}

Write-Host ''
"{0,-20} {1,10}" -f '产物', '体积(MB)' | Write-Host
foreach ($f in $results) {
    $size = (Get-Item $f.FullName).Length / 1MB
    "{0,-20} {1,10:N2}" -f $f.Name, $size | Write-Host
}
