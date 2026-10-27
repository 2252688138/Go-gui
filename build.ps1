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
    [ValidateSet('gallery', 'demo', 'tools', 'all')]
    [string]$Target = 'gallery',
    [switch]$Upx,
    [switch]$Dev,
    [string]$OutDir = 'dist'
)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)

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
