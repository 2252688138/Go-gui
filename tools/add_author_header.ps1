# 为 Go / 汇编源文件批量添加作者信息头
# 用法：powershell -NoProfile -ExecutionPolicy Bypass -File tools\add_author_header.ps1 [-Path internal,ui,cmd] [-DryRun]
param(
  [string[]]$Path = @('internal', 'ui', 'cmd'),
  [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
Set-Location $repo

$HeaderLines = @(
  ' Go-gui',
  ' Author  : Mr.Shu',
  ' Website : www.cq6k.com',
  ' QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6'
)

$changed = 0
$skipped = 0

foreach ($root in $Path) {
  if (-not (Test-Path $root)) { continue }
  $files = Get-ChildItem -Path $root -Recurse -File -Include *.go, *.s
  foreach ($f in $files) {
    $raw = [IO.File]::ReadAllText($f.FullName)
    if ($raw -match 'Mr\.Shu') { $skipped++; continue }

    $hasBom = $raw.Length -gt 0 -and $raw[0] -eq [char]0xFEFF
    $text = if ($hasBom) { $raw.Substring(1) } else { $raw }
    $eol = if ($text -match "`r`n") { "`r`n" } else { "`n" }

    # 按行切分（保留行尾），定位插入点：build tag 块之后，否则文件开头
    $lines = [regex]::Split($text, "(?<=`n)")
    $idx = 0
    if ($lines.Count -gt 0 -and $lines[0] -match '^//go:build') {
      while ($idx -lt $lines.Count -and $lines[$idx] -match '^//(go:build| \+build)') { $idx++ }
      if ($idx -lt $lines.Count -and $lines[$idx].Trim() -eq '') { $idx++ }
    }

    $header = ''
    foreach ($h in $HeaderLines) { $header += "//$h$eol" }
    $header += $eol

    $newText = ($lines[0..($idx - 1)] -join '') + $header + ($lines[$idx..($lines.Count - 1)] -join '')
    if ($idx -eq 0) { $newText = $header + $text }

    if ($DryRun) {
      "DRY: $($f.FullName.Substring($repo.Length + 1)) (insert at line $($idx + 1))"
    } else {
      $enc = New-Object System.Text.UTF8Encoding($hasBom)
      [IO.File]::WriteAllText($f.FullName, $newText, $enc)
      $changed++
    }
  }
}

"changed=$changed skipped=$skipped"
