<#
.SYNOPSIS
GoGUI Android 构建脚本：把 mobile/android 编译成 libgogui.so（NativeActivity 直接加载）。

.DESCRIPTION
  产物：dist/android/<abi>/libgogui.so
  需要：Android NDK（自动探测 ANDROID_NDK_HOME / ANDROID_NDK_ROOT / <SDK>\ndk\*）
  说明：Android 端使用 cgo（移动端唯一可行路径），因此必须 CGO_ENABLED=1 且指定 NDK 的 clang。

.EXAMPLE
./build-android.ps1                    # arm64-v8a + armeabi-v7a
./build-android.ps1 -Abi arm64-v8a     # 只构建 64 位
./build-android.ps1 -Package           # 额外调用 gradle 打包 APK（需 SDK + gradle）
#>
param(
    [ValidateSet('arm64-v8a', 'armeabi-v7a', 'all')]
    [string]$Abi = 'all',
    [switch]$Package,
    [string]$OutDir = 'dist/android'
)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)

function Find-Ndk {
    foreach ($p in @($env:ANDROID_NDK_HOME, $env:ANDROID_NDK_ROOT)) {
        if ($p -and (Test-Path $p)) { return $p }
    }
    $sdk = $env:ANDROID_HOME
    if (-not $sdk) { $sdk = "$env:LOCALAPPDATA\Android\Sdk" }
    if (Test-Path "$sdk\ndk") {
        $latest = Get-ChildItem "$sdk\ndk" -Directory | Sort-Object Name -Descending | Select-Object -First 1
        if ($latest) { return $latest.FullName }
    }
    return $null
}

$ndk = Find-Ndk
if (-not $ndk) {
    throw '未找到 Android NDK：请安装 NDK（SDK Manager）并设置 ANDROID_NDK_HOME'
}
Write-Host "[ndk] $ndk" -ForegroundColor Cyan

$hostTag = 'windows-x86_64'
$binDir = Join-Path $ndk "toolchains\llvm\prebuilt\$hostTag\bin"
if (-not (Test-Path $binDir)) {
    throw "找不到 NDK 工具链目录：$binDir（非 Windows 主机请改用对应 prebuilt 目录）"
}

# Go 的 GOARCH → NDK clang 前缀
$targets = @(
    @{ Abi = 'arm64-v8a';   GOARCH = 'arm64'; Prefix = 'aarch64-linux-android21-clang.cmd' },
    @{ Abi = 'armeabi-v7a'; GOARCH = 'arm'  ; Prefix = 'armv7a-linux-androideabi21-clang.cmd' }
)
if ($Abi -ne 'all') {
    $targets = $targets | Where-Object { $_.Abi -eq $Abi }
}

$results = @()
foreach ($t in $targets) {
    $cc = Join-Path $binDir $t.Prefix
    if (-not (Test-Path $cc)) {
        Write-Host "[skip] 缺少编译器 $cc" -ForegroundColor Yellow
        continue
    }
    $abiDir = Join-Path $OutDir $t.Abi
    New-Item -ItemType Directory -Force -Path $abiDir | Out-Null
    $so = Join-Path $abiDir 'libgogui.so'

    Write-Host ("[build] {0}  GOARCH={1}" -f $t.Abi, $t.GOARCH) -ForegroundColor Cyan
    $env:CC = $cc
    $env:CGO_ENABLED = '1'
    $env:GOOS = 'android'
    $env:GOARCH = $t.GOARCH
    $env:GOARM = '7'

    & go build -buildmode=c-shared -trimpath -ldflags '-s -w' -o $so ./mobile/android
    if ($LASTEXITCODE -ne 0) { throw "构建失败：$($t.Abi)" }
    $results += (Get-Item -LiteralPath $so)
}

# 清理 c-shared 生成的 .h（App 不需要）
Get-ChildItem $OutDir -Recurse -Filter '*.h' -ErrorAction SilentlyContinue | Remove-Item -Force

Write-Host ''
'{0,-34} {1,10}' -f '产物', '体积(MB)' | Write-Host
'-' * 46 | Write-Host
foreach ($f in $results) {
    '{0,-34} {1,10:N2}' -f $f.FullName.Replace((Get-Location).Path + '\', ''), ($f.Length / 1MB) | Write-Host
}
Write-Host ''

if ($Package) {
    $gradle = Get-Command gradle -ErrorAction SilentlyContinue
    if (-not $gradle) {
        Write-Host '未找到 gradle：请安装 Gradle 或用 Android Studio 打开 mobile/android' -ForegroundColor Yellow
    } else {
        Push-Location mobile/android
        try { & $gradle.Source assembleDebug } finally { Pop-Location }
    }
}
