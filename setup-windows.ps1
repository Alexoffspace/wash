#requires -Version 5.1

$Script:ProjectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Script:ProjectDir

# ─── ANSI ──────────────────────────────────────────
$R = "$([char]27)[0;31m"; $G = "$([char]27)[0;32m"
$C = "$([char]27)[0;36m"; $Y = "$([char]27)[1;33m"
$B = "$([char]27)[1m"; $D = "$([char]27)[2m"
$N = "$([char]27)[0m"
$HIDE = "$([char]27)[?25l"; $SHOW = "$([char]27)[?25h"
$CLR = "$([char]27)[2J"; $HOME1 = "$([char]27)[H"

$CHECK_ON  = [char]0x25C9  # ◉
$CHECK_OFF = [char]0x25CB  # ○

# ─── State ─────────────────────────────────────────
$Script:FieldTypes  = @("bool","str","bool","str","str","bool","str","str","bool")
$Script:FieldNames  = @("os_auth","token_env","set_token","token_value","port","allow_0","shell","work_dir","install_service")
$Script:FieldLabels = @(
  "OS Authentication","Token env var","Set token value in .env","Token value",
  "Port","Listen on 0.0.0.0","Shell command","Work directory",
  "Install as Windows service"
)
$Script:FieldCount = $Script:FieldTypes.Count
$Script:BtnSave    = $Script:FieldCount
$Script:BtnCancel  = $Script:FieldCount + 1
$Script:TotalItems = $Script:FieldCount + 2

$Script:Defaults = @("true","WASH_TOKEN","false","","9091","false","sh","","false")
$Script:Values   = $Script:Defaults.Clone()

$Script:RowBase = 5
$Script:ColVal  = 28
$Script:ValW    = 30
$Script:RowBtn  = $Script:RowBase + $Script:FieldCount + 2
$Script:RowStat = $Script:RowBtn + 2

$Script:GoOk = $false
$Script:CleanExit = $false

# ─── Console helpers ──────────────────────────────
function Cursor($r, $c) { [Console]::SetCursorPosition($c, $r) }
function ClrEol { $w = [Console]::WindowWidth; $p = [Console]::CursorTop; $c = [Console]::CursorLeft; Write-Host (" " * ($w - $c)) -NoNewline; Cursor $p $c }
function ClrScr { Write-Host "${CLR}${HOME1}" -NoNewline }

function RawOn  { }
function RawOff { }

# ─── Helpers ──────────────────────────────────────
function GetFieldType($i) {
  if ($i -lt $Script:FieldCount) { return $Script:FieldTypes[$i] }
  return "btn"
}

function GetFieldName($i) {
  if ($i -ge $Script:FieldCount) { return "" }
  return $Script:FieldNames[$i]
}

function GetFieldLabel($i) {
  if ($i -ge $Script:FieldCount) { return "" }
  return $Script:FieldLabels[$i]
}

function IsVisible($i) {
  if ($i -ge $Script:FieldCount) { return $true }
  $n = GetFieldName $i
  if ($n -eq "token_value" -and $Script:Values[2] -ne "true") { return $false }
  return $true
}

function VisibleIndices {
  $out = @()
  for ($i = 0; $i -lt $Script:TotalItems; $i++) {
    if (IsVisible $i) { $out += $i }
  }
  return $out
}

# ─── TUI draw ─────────────────────────────────────
function DrawStatic {
  ClrScr
  Write-Host "${HIDE}" -NoNewline
  Write-Host ""
  Write-Host "  ${B}WASH${N} ${D}— Setup & Build${N}"
  Write-Host ""
  Write-Host ""
  Write-Host "  ${D}Navigation: ↑↓ / Tab · Space: toggle · Enter: edit/confirm${N}"
  Write-Host ""
  Write-Host ""

  for ($i = 0; $i -lt $Script:FieldCount; $i++) {
    $row = $Script:RowBase + $i
    Cursor $row 0; ClrEol
    $type = GetFieldType $i
    $label = GetFieldLabel $i
    if ($type -eq "bool") {
      Write-Host "  ${CHECK_OFF}  ${label}" -NoNewline
    } else {
      Write-Host "  ${label}:" -NoNewline
    }
  }

  Cursor $Script:RowBtn 0; ClrEol
  Write-Host "      ${D}< Save >${N}           ${D}< Cancel >${N}" -NoNewline

  Cursor $Script:RowStat 0; ClrEol
  Write-Host "  ${D}Ready — choose Save to build, or Cancel to exit${N}" -NoNewline
}

function DrawItem($i) {
  if (-not (IsVisible $i)) {
    $row = $Script:RowBase + $i
    Cursor $row 0; ClrEol
    return
  }

  $type = GetFieldType $i
  $label = GetFieldLabel $i
  $val = $Script:Values[$i]

  if ($i -ge $Script:FieldCount) { return }

  $row = $Script:RowBase + $i
  Cursor $row 0
  if ($type -eq "bool") {
    $chk = $CHECK_OFF
    if ($val -eq "true") { $chk = $CHECK_ON }
    Write-Host "  ${B}${chk}${N}  ${label}" -NoNewline; ClrEol
  } else {
    $display = $val
    if ((GetFieldName $i) -eq "token_value" -and -not [string]::IsNullOrEmpty($val)) {
      $display = "********"
    }
    Write-Host "  ${label}: " -NoNewline
    Write-Host "${B}[${N}$($display.PadRight($Script:ValW).Substring(0,[Math]::Min($display.Length,$Script:ValW)))${B}]${N}" -NoNewline; ClrEol
  }
}

function HighlightItem($i) {
  if (-not (IsVisible $i)) { return }

  $type = GetFieldType $i
  $label = GetFieldLabel $i
  $val = $Script:Values[$i]

  if ($i -ge $Script:FieldCount) {
    Cursor $Script:RowBtn 0; ClrEol
    if ($i -eq $Script:BtnSave) {
      Write-Host "      ${B}${G}< Save >${N}${D}           ${B}< Cancel >${N}" -NoNewline
    } else {
      Write-Host "      ${D}${B}< Save >${N}           ${B}${G}< Cancel >${N}" -NoNewline
    }
    return
  }

  $row = $Script:RowBase + $i
  Cursor $row 0
  if ($type -eq "bool") {
    $chk = $CHECK_OFF
    if ($val -eq "true") { $chk = $CHECK_ON }
    Write-Host "  ${B}${G}${chk}${N} ${B}${G}${label}${N}" -NoNewline; ClrEol
  } else {
    $display = $val
    if ((GetFieldName $i) -eq "token_value" -and -not [string]::IsNullOrEmpty($val)) {
      $display = "********"
    }
    Write-Host "  ${B}${G}${label}${N}${B}${G}: ${N}" -NoNewline
    Write-Host "${B}${G}[${N}${G}$($display.PadRight($Script:ValW).Substring(0,[Math]::Min($display.Length,$Script:ValW)))${N}${B}${G}]${N}" -NoNewline; ClrEol
  }

  Cursor $Script:RowBtn 0; ClrEol
  Write-Host "      ${D}< Save >${N}           ${D}< Cancel >${N}" -NoNewline
}

function Status($msg, $color) {
  if (-not $color) { $color = $C }
  Cursor $Script:RowStat 0; ClrEol
  Write-Host "  ${color}${msg}${N}" -NoNewline
}

# ─── Input loop ───────────────────────────────────
function TuiLoop {
  DrawStatic
  for ($i = 0; $i -lt $Script:FieldCount; $i++) { DrawItem $i }

  $vis = VisibleIndices
  $cur = $vis[0]
  HighlightItem $cur
  Status "Ready"

  $editing = $false
  $editOld = ""

  while ($true) {
    $keyInfo = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    $ch  = $keyInfo.Character
    $key = $keyInfo.Key
    $mod = $keyInfo.Modifiers

    # Determine action
    $action = ""
    $isPrintable = $false

    if ($key -eq "UpArrow")    { $action = "UP" }
    elseif ($key -eq "DownArrow")  { $action = "DOWN" }
    elseif ($key -eq "LeftArrow")  { $action = "LEFT" }
    elseif ($key -eq "RightArrow") { $action = "RIGHT" }
    elseif ($key -eq "Tab")        { $action = "DOWN" }
    elseif ($key -eq "Enter")      { $action = "ENTER" }
    elseif ($key -eq "Spacebar")   { $action = "SPACE" }
    elseif ($key -eq "Escape")     { $action = "ESC" }
    elseif ($key -eq "Backspace")  { $action = "BACKSPACE" }
    elseif ($ch -ne 0) {
      $isPrintable = $true
    }

    # current visible indices
    $vis = VisibleIndices
    $visLen = $vis.Count
    $vi = -1
    for ($v = 0; $v -lt $visLen; $v++) { if ($vis[$v] -eq $cur) { $vi = $v; break } }
    if ($vi -eq -1) { $cur = $vis[0]; $vi = 0 }

    if ($editing) {
      # ── Edit mode ──
      $fname = GetFieldName $cur
      if ($action -eq "ENTER") {
        if ($fname -eq "port") {
          $pv = $Script:Values[$cur]
          if (-not ($pv -match '^\d+$') -or [int]$pv -lt 1 -or [int]$pv -gt 65535) {
            Status "Port must be 1–65535" $R
            $Script:Values[$cur] = $editOld
            DrawItem $cur
            HighlightItem $cur
            continue
          }
        }
        $editing = $false
        DrawItem $cur
        $nvi = $vi + 1
        if ($nvi -ge $visLen) { $nvi = 0 }
        $cur = $vis[$nvi]
        HighlightItem $cur
        Status "Ready"
      } elseif ($action -eq "ESC") {
        $Script:Values[$cur] = $editOld
        $editing = $false
        DrawItem $cur
        HighlightItem $cur
        Status "Cancelled"
      } elseif ($action -eq "BACKSPACE") {
        if ($Script:Values[$cur].Length -gt 0) {
          $Script:Values[$cur] = $Script:Values[$cur].Substring(0, $Script:Values[$cur].Length - 1)
        }
        HighlightItem $cur
      } elseif ($isPrintable) {
        $Script:Values[$cur] += $ch
        HighlightItem $cur
      }
    } else {
      # ── Navigation mode ──
      switch ($action) {
        "UP" {
          DrawItem $cur
          $pvi = $vi - 1
          if ($pvi -lt 0) { $pvi = $visLen - 1 }
          $cur = $vis[$pvi]
          HighlightItem $cur
        }
        "DOWN" {
          DrawItem $cur
          $nvi = $vi + 1
          if ($nvi -ge $visLen) { $nvi = 0 }
          $cur = $vis[$nvi]
          HighlightItem $cur
        }
        "SPACE" {
          $ftype = GetFieldType $cur
          if ($ftype -eq "bool") {
            if ($Script:Values[$cur] -eq "true") { $Script:Values[$cur] = "false" } else { $Script:Values[$cur] = "true" }
            DrawItem $cur
            HighlightItem $cur
            $fname = GetFieldName $cur
            if ($fname -eq "set_token") {
              if ($Script:Values[2] -eq "true") {
                DrawItem 3
              } else {
                $tvRow = $Script:RowBase + 3
                Cursor $tvRow 0; ClrEol
              }
            }
          }
        }
        "ENTER" {
          $ftype = GetFieldType $cur
          if ($cur -ge $Script:FieldCount) {
            if ($cur -eq $Script:BtnSave) {
              SaveAndBuild
              return
            } else {
              $Script:CleanExit = $true
              ClrScr
              Write-Host "${SHOW}" -NoNewline
              Write-Host ""
              Write-Host "  Setup cancelled."
              Write-Host ""
              exit 0
            }
          } elseif ($ftype -eq "bool") {
            if ($Script:Values[$cur] -eq "true") { $Script:Values[$cur] = "false" } else { $Script:Values[$cur] = "true" }
            DrawItem $cur
            HighlightItem $cur
            $fname = GetFieldName $cur
            if ($fname -eq "set_token") {
              if ($Script:Values[2] -eq "true") {
                DrawItem 3
              } else {
                $tvRow = $Script:RowBase + 3
                Cursor $tvRow 0; ClrEol
              }
            }
          } else {
            $editOld = $Script:Values[$cur]
            $editing = $true
            Status "Editing... Enter=confirm, Esc=cancel" $Y
            HighlightItem $cur
          }
        }
        default {
          if ($isPrintable) {
            $ftype = GetFieldType $cur
            if ($ftype -eq "str" -and $cur -lt $Script:FieldCount) {
              $fname = GetFieldName $cur
              if ($fname -ne "token_value" -or $Script:Values[2] -eq "true") {
                $editOld = $Script:Values[$cur]
                $Script:Values[$cur] = $ch
                $editing = $true
                Status "Editing... Enter=confirm, Esc=cancel" $Y
                HighlightItem $cur
              }
            }
          }
        }
      }
    }
  }
}

# ─── Go ─────────────────────────────────────────────
function Check-Go {
  $goCmd = Get-Command "go" -ErrorAction SilentlyContinue
  if ($goCmd) { $Script:GoOk = $true; return $true }
  return $false
}

function Install-Go {
  Write-Host ""
  Write-Host "  ${C}Go not found. Installing latest Go...${N}"
  Write-Host ""

  $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

  # Get latest version
  $goVer = ""
  try {
    $resp = Invoke-WebRequest -Uri "https://go.dev/VERSION?m=text" -UseBasicParsing -TimeoutSec 10
    $goVer = ($resp.Content -split "`n")[0]
  } catch {}
  if ([string]::IsNullOrEmpty($goVer)) { $goVer = "go1.24.0" }
  $goVer = $goVer -replace "^go", ""

  $msiUrl = "https://go.dev/dl/go${goVer}.windows-${arch}.msi"
  $msiPath = "$env:TEMP\go.msi"

  Write-Host "  ${D}Downloading go${goVer} for Windows (${arch})...${N}"
  try {
    Invoke-WebRequest -Uri $msiUrl -OutFile $msiPath -UseBasicParsing
  } catch {
    Write-Host "  ${R}Download failed: $_${N}"
    return $false
  }

  Write-Host "  ${D}Installing Go (MSI)...${N}"
  try {
    Start-Process msiexec.exe -Wait -ArgumentList "/i `"$msiPath`" /quiet"
  } catch {
    Write-Host "  ${R}Installation failed: $_${N}"
    return $false
  }

  # Add to PATH
  $goBin = "${env:ProgramFiles}\Go\bin"
  $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
  if ($userPath -notlike "*${goBin}*") {
    [Environment]::SetEnvironmentVariable("PATH", "$userPath;$goBin", "User")
  }
  $env:PATH += ";${goBin}"

  Remove-Item -Force $msiPath -ErrorAction SilentlyContinue
  $Script:GoOk = $true
  Write-Host ""
  Write-Host "  ${G}Go ${goVer} installed.${N}"
  return $true
}

# ─── Windows Service ────────────────────────────────
function Install-WindowsService {
  $svcDir = $PWD.Path
  $binPath = Join-Path $svcDir "WASH.exe"

  Write-Host ""
  Write-Host "  ${C}Installing Windows service...${N}"
  Write-Host ""

  # Stop & remove existing service
  $existing = Get-Service -Name "WASH" -ErrorAction SilentlyContinue
  if ($existing) {
    Write-Host "  ${D}Stopping existing WASH service...${N}"
    Stop-Service -Name "WASH" -Force -ErrorAction SilentlyContinue
    sc.exe delete "WASH" 2>&1 | Out-Null
  }

  # Create new service
  $envFile = Join-Path $svcDir ".env"
  $envArg = ""
  if (Test-Path $envFile) {
    $envArg = " --env-file `"$envFile`""
  }

  New-Service -Name "WASH" `
    -BinaryPathName "`"$binPath`" --service${envArg}" `
    -DisplayName "WASH (Web Accessible Shell)" `
    -Description "Web-based shell terminal with REST API and WebSocket support" `
    -StartupType Automatic | Out-Null

  # Configure failure recovery
  sc.exe failure "WASH" reset=86400 actions=restart/5000/restart/10000/restart/30000 2>&1 | Out-Null

  Start-Service -Name "WASH"

  Write-Host "  $([char]0x2713) ${G}Windows service installed and started${N}"
  Write-Host "  ${D}Manage: Get-Service WASH | Start-Service / Stop-Service / Restart-Service${N}"
}

# ─── Save & Build ─────────────────────────────────
function SaveAndBuild {
  RawOff
  Write-Host "${SHOW}${CLR}${HOME1}" -NoNewline

  Write-Host ""
  Write-Host "  ${B}${C}── Saving configuration ──${N}"
  Write-Host ""

  # Validate port
  $portVal = $Script:Values[4]
  if (-not ($portVal -match '^\d+$') -or [int]$portVal -lt 1 -or [int]$portVal -gt 65535) {
    Write-Host "  ${R}Error: Port must be 1–65535${N}"
    Write-Host ""
    Write-Host "  ${D}Press Enter to go back...${N}"
    $Host.UI.RawUI.ReadKey("IncludeKeyDown") | Out-Null
    DrawStatic
    for ($i = 0; $i -lt $Script:FieldCount; $i++) { DrawItem $i }
    $vis = VisibleIndices
    $cur = $vis[0]
    HighlightItem $cur
    Status "Ready"
    return
  }

  # config.yaml
@"
# WASH (Web Accessible Shell) configuration file
# Generated by setup-windows.ps1

# Enable OS authentication (true/false)
os_auth: $($Script:Values[0])

# Environment variable name for the token
token: $($Script:Values[1])

# Port on which the application will run
port: $($Script:Values[4])

# Listen on 0.0.0.0 (true) or 127.0.0.1 (false)
allow_0: $($Script:Values[5])

# Working directory (empty = user home)
work_dir: $($Script:Values[7])

# Shell command for interactive sessions
# Windows examples:
#   powershell.exe   — built-in Windows PowerShell
#   pwsh             — PowerShell 7+
#   cmd              — legacy Command Prompt
#   C:\Program Files\Git\bin\bash.exe — Git Bash
shell: $($Script:Values[6])
"@ | Out-File -FilePath config.yaml -Encoding utf8

  Write-Host "  $([char]0x2713) ${G}config.yaml written${N}"

  # .env
  if ($Script:Values[2] -eq "true" -and -not [string]::IsNullOrEmpty($Script:Values[3])) {
@"
# WASH environment variables
# Generated by setup-windows.ps1

$($Script:Values[1])=$($Script:Values[3])
"@ | Out-File -FilePath .env -Encoding utf8

    Write-Host "  $([char]0x2713) ${G}.env written with token${N}"
  }

  Write-Host ""
  Write-Host "  ${B}${C}── Building WASH ──${N}"
  Write-Host ""

  if (-not $Script:GoOk) {
    if (-not (Check-Go)) {
      Write-Host "  ${R}Error: Go is not installed.${N}"
      Write-Host "  Run the script again and allow Go installation."
      Write-Host ""
      Write-Host "  ${D}Press Enter to go back...${N}"
      $Host.UI.RawUI.ReadKey("IncludeKeyDown") | Out-Null
      DrawStatic
      for ($i = 0; $i -lt $Script:FieldCount; $i++) { DrawItem $i }
      $vis = VisibleIndices
      $cur = $vis[0]
      HighlightItem $cur
      Status "Ready"
      return
    }
  }

  go build -o WASH.exe .
  Write-Host ""
  Write-Host "  $([char]0x2713) ${G}${B}Build successful!${N}  Binary: ${B}WASH.exe${N}"

  if ($Script:Values[8] -eq "true") {
    Install-WindowsService
  }

  Write-Host ""
  Write-Host "  ${D}Run:${N}"
  Write-Host "    ${B}.\WASH.exe -token=YOUR_TOKEN -port=9091${N}"
  Write-Host "    ${B}.\WASH.exe -os-auth -port=9091${N}"
  Write-Host ""
  Write-Host "  ${D}Edit config.yaml for permanent settings.${N}"
  Write-Host ""
  Write-Host "  ${D}Press Enter to exit...${N}"
  $Host.UI.RawUI.ReadKey("IncludeKeyDown") | Out-Null

  $Script:CleanExit = $true
  ClrScr
  Write-Host "${SHOW}" -NoNewline
  Write-Host ""
  exit 0
}

# ─── Main ──────────────────────────────────────────
Write-Host "${CLR}${HOME1}${SHOW}" -NoNewline
Write-Host ""
Write-Host "  ${B}WASH — Setup & Build${N}"
Write-Host "  ${D}━━━━━━━━━━━━━━━━━━━━━${N}"
Write-Host ""

if (-not (Check-Go)) {
  Install-Go
} else {
  $goVer = &go version
  Write-Host "  $([char]0x2713) ${G}Go $($goVer -replace '.*go(\S+).*','$1') found${N}"
}
Write-Host ""

Write-Host "  ${D}Starting configuration TUI...${N}"
Start-Sleep -Milliseconds 500

TuiLoop
