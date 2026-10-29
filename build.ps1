<#
.SYNOPSIS
GoGUI 构建脚本：版本号注入 + 体积优化构建 + Windows 资源生成 + NSIS 安装器打包。

.DESCRIPTION
版本号管理
  仓库根目录的 VERSION.txt 是唯一来源（语义化版本 x.y.z）。
  脚本读取后做三件事：
    1. 生成 cmd/gallery/rsrc_windows_*.syso（图标 + 文件属性版本，见 tools/winres）
    2. 通过 -ldflags -X 注入 gogui/internal/version.*（运行时 version.String() / version.Full()）
    3. 传给 NSIS 作为安装器版本与「应用和功能」里的显示版本
  改版本：编辑 VERSION.txt，或用 -Bump patch|minor|major 自动递增。

发布构建参数
  -trimpath -buildvcs=false                去掉本机绝对路径与 VCS 信息（可复现、体积更小）
  -ldflags "-s -w -H windowsgui -buildid=" 去掉符号表/调试信息/构建 ID，GUI 子系统不弹控制台
  CGO_ENABLED=0                            纯 syscall 实现，无 cgo 依赖，产物更小且免运行库
  GOAMD64=v1                               保守指令集，兼容旧 CPU（自用机器可改 v3 换性能）

实测（gallery，go1.26）：
  默认 go build            6.45 MB
  本脚本发布构建           4.54 MB   (-30%)
  本脚本发布构建 + -Upx    1.52 MB   (-76%)

.EXAMPLE
./build.ps1                          # 发布构建 gallery -> dist\gallery.exe
./build.ps1 -Installer               # 构建 + 打 NSIS 安装器 -> dist\GoGUI-0.1.0-setup.exe
./build.ps1 -Installer -Upx          # 再叠加 UPX -9（体积最小）
./build.ps1 -Bump patch -Installer   # 版本号 +1 后打包（0.1.0 -> 0.1.1）
./build.ps1 -Channel release -Installer   # 正式通道构建（exe 属性不带 DEBUG 标记）
./build.ps1 -Target all              # gallery + demo + 三个调试工具
./build.ps1 -Dev                     # 开发构建：保留符号、带控制台（stderr 日志可见）
./build.ps1 -NoRes                   # 跳过 syso 生成（沿用已有资源）
#>
param(
    [ValidateSet('gallery', 'demo', 'tools', 'all', 'web')]
    [string]$Target = 'gallery',
    [switch]$Upx,
    [switch]$Dev,
    [string]$OutDir = 'dist',
    # 仅 web 目标：构建完启动本地静态服务器并打开浏览器
    [switch]$Serve,
    [int]$Port = 8088,
    # ---- 版本号管理 ----
    [ValidateSet('none', 'patch', 'minor', 'major')]
    [string]$Bump = 'none',
    [string]$VersionOverride = '',
    [ValidateSet('dev', 'beta', 'release')]
    [string]$Channel = 'dev',
    # ---- Windows 资源与安装器 ----
    [switch]$NoRes,
    [switch]$Installer,
    [string]$ProductName = 'GoGUI',
    [string]$Company = 'Go-gui',
    [string]$Description = 'GoGUI 纯 Go 自绘 GUI 框架',
    [string]$Copyright = 'Copyright (c) 2026 Mr.Shu',
    [string]$AppExe = 'gallery.exe',
    # 图标：exe 资源与安装器共用
    [string]$IconFile = 'ui\assets\icon.ico'
)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)

# ================= 版本号：VERSION.txt 是唯一来源 =================
$versionFile = Join-Path $PSScriptRoot 'VERSION.txt'

function Get-ProjectVersion {
    param([string]$File)
    if (-not (Test-Path -LiteralPath $File)) { throw "找不到版本文件：$File" }
    $raw = [IO.File]::ReadAllText($File).Trim().TrimStart([char]0xFEFF).Trim()
    if ($raw -notmatch '^v?(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.\-]+)?$') {
        throw "VERSION.txt 内容不合法：'$raw'（应为 x.y.z，如 1.2.3）"
    }
    return $raw.TrimStart('v', 'V')
}

$Version = if ($VersionOverride) { $VersionOverride.Trim().TrimStart('v', 'V') } else { Get-ProjectVersion $versionFile }

if ($Bump -ne 'none') {
    if ($Version -notmatch '^(\d+)\.(\d+)\.(\d+)') { throw "无法递增版本号：'$Version'" }
    $maj = [int]$Matches[1]; $min = [int]$Matches[2]; $pat = [int]$Matches[3]
    switch ($Bump) {
        'major' { $maj++; $min = 0; $pat = 0 }
        'minor' { $min++; $pat = 0 }
        'patch' { $pat++ }
    }
    $Version = "$maj.$min.$pat"
    [IO.File]::WriteAllText($versionFile, "$Version`n")
    Write-Host "[version] 版本号递增 -> $Version（已写回 VERSION.txt）" -ForegroundColor Yellow
}

# 4 段版本号（安装器 VIProductVersion 要求 x.x.x.x）
$viVersion = if ($Version -match '^(\d+)\.(\d+)\.(\d+)') { "$($Matches[1]).$($Matches[2]).$($Matches[3]).0" } else { '0.0.0.0' }

# 构建元信息
$Commit = 'none'
try {
    $gitCommit = (& git rev-parse --short HEAD 2>$null)
    if ($LASTEXITCODE -eq 0 -and $gitCommit) { $Commit = $gitCommit.Trim() }
} catch { $Commit = 'none' }
$BuildTime = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

Write-Host ""
Write-Host ("[version] {0} {1} · {2} · {3} · commit {4}" -f $ProductName, $Version, $Channel, $BuildTime, $Commit) -ForegroundColor Green

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

# ================= Windows 资源（图标 + 文件属性版本）=================
# tools/winres 生成 COFF 目标文件，Go 链接器把 .rsrc 节合并进 PE（无需 windres/rsrc）
$iconPath = Join-Path $PSScriptRoot $IconFile
$needRes = (-not $NoRes) -and ($Target -in @('gallery', 'all'))

if ($needRes -and -not (Test-Path -LiteralPath $iconPath)) {
    Write-Warning "图标不存在，跳过资源生成：$iconPath"
    $needRes = $false
}

if ($needRes) {
    foreach ($arch in @('amd64', '386')) {
        $syso = Join-Path $PSScriptRoot ("cmd\gallery\rsrc_windows_{0}.syso" -f $arch)
        Write-Host ("[res] rsrc_windows_{0}.syso  (v{1}, {2})" -f $arch, $Version, $Channel) -ForegroundColor Cyan
        $winresArgs = @(
            'run', './tools/winres',
            '-arch', $arch,
            '-ico', $iconPath,
            '-out', $syso,
            '-version', $Version,
            '-product', $ProductName,
            '-company', $Company,
            '-desc', $Description,
            '-copyright', $Copyright,
            '-commit', $Commit,
            '-channel', $Channel,
            '-q'
        )
        & go @winresArgs
        if ($LASTEXITCODE -ne 0) { throw "生成 Windows 资源失败：$arch" }
    }
}

# ================= 构建 =================
# 运行时版本信息（gogui/internal/version.*）通过链接期变量注入
$versionLd = @(
    "-X gogui/internal/version.Number=$Version",
    "-X gogui/internal/version.Commit=$Commit",
    "-X gogui/internal/version.BuildTime=$BuildTime",
    "-X gogui/internal/version.Channel=$Channel",
    "-X gogui/internal/version.Product=$ProductName",
    "-X gogui/internal/version.Company=$Company"
) -join ' '

if ($Dev) {
    $ldArgs = "-H windowsgui $versionLd"
    $extra = @('-buildvcs=false')
} else {
    $ldArgs = "-s -w -H windowsgui -buildid= $versionLd"
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

# ================= NSIS 安装器 =================
function Find-Makensis {
    $candidates = @()
    if ($env:NSIS_EXE) { $candidates += $env:NSIS_EXE }
    $candidates += @(
        'C:\Program Files (x86)\NSIS\makensis.exe',
        'C:\Program Files\NSIS\makensis.exe',
        'H:\JIMaX\bin\makensis.exe',
        '.\bin\makensis.exe'
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path -LiteralPath $c)) { return $c }
    }
    $cmd = Get-Command makensis -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return $null
}

if ($Installer) {
    $builtExe = Join-Path $OutDir $AppExe
    if (-not (Test-Path -LiteralPath $builtExe)) {
        throw "找不到待打包的程序：$builtExe（请用 -Target gallery/all 构建）"
    }

    $makensis = Find-Makensis
    if (-not $makensis) {
        throw '未找到 makensis.exe：请安装 NSIS 3（https://nsis.sourceforge.io/），或用环境变量 NSIS_EXE 指定路径'
    }

    $setupName = '{0}-{1}-setup.exe' -f $ProductName, $Version
    $setupPath = Join-Path (Resolve-Path $OutDir) $setupName
    $nsiPath = Join-Path $PSScriptRoot 'installer\gogui.nsi'

    Write-Host ("[nsis] {0}  ->  {1}" -f $nsiPath, $setupPath) -ForegroundColor Cyan
    $nsisArgs = @(
        '/V2',
        "/DPRODUCT=$ProductName",
        "/DVERSION=$Version",
        "/DVI_VERSION=$viVersion",
        "/DCOMPANY=$Company",
        "/DDESC=$Description",
        "/DCOPYRIGHT=$Copyright",
        "/DCHANNEL=$Channel",
        "/DAPPEXE=$AppExe",
        "/DSRC_EXE=$((Resolve-Path $builtExe).Path)",
        "/DOUT_FILE=$setupPath",
        "/DICON_FILE=$iconPath",
        $nsiPath
    )
    & $makensis @nsisArgs
    if ($LASTEXITCODE -ne 0) { throw "makensis 打包失败（退出码 $LASTEXITCODE）" }

    if (Test-Path -LiteralPath $setupPath) {
        $inst = Get-Item $setupPath
        $results += $inst
    }
}

# ================= 汇总 =================
Write-Host ''
"{0,-34} {1,10}" -f '产物', '体积(MB)' | Write-Host
foreach ($f in $results) {
    $size = (Get-Item $f.FullName).Length / 1MB
    "{0,-34} {1,10:N2}" -f $f.Name, $size | Write-Host
}
Write-Host ''
Write-Host ("版本 {0} · 通道 {1} · 提交 {2}" -f $Version, $Channel, $Commit) -ForegroundColor Green
if ($Installer) {
    Write-Host ("安装器 {0}" -f (Join-Path (Resolve-Path $OutDir) ('{0}-{1}-setup.exe' -f $ProductName, $Version))) -ForegroundColor Green
}
