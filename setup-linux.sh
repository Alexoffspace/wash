#!/usr/bin/env bash
set -euo pipefail

cd "$(cd "$(dirname "$0")" && pwd)"

# ─── ANSI ──────────────────────────────────────────
R='\033[0;31m'; G='\033[0;32m'; C='\033[0;36m'
Y='\033[1;33m'; B='\033[1m'; D='\033[2m'; N='\033[0m'
CLR='\033[2J'; HOME1='\033[H'; HIDE='\033[?25l'; SHOW='\033[?25h'

CHECK_ON="$(printf '\xe2\x97\x89')"   # ◉
CHECK_OFF="$(printf '\xe2\x97\x8b')"  # ○

# ─── State ─────────────────────────────────────────
FIELDS=(
  "bool:os_auth:OS Authentication"
  "str:token_env:Token env var"
  "bool:set_token:Set token value in .env"
  "str:token_value:Token value"
  "str:port:Port"
  "bool:allow_0:Listen on 0.0.0.0"
  "str:shell:Shell command"
  "str:work_dir:Work directory"
  "bool:install_service:Install as system service"
  "str:service_user:Service user"
)
FIELD_COUNT=${#FIELDS[@]}
BTN_SAVE=$FIELD_COUNT
BTN_CANCEL=$((FIELD_COUNT + 1))
TOTAL=$((FIELD_COUNT + 2))

DEFAULTS=("true" "WASH_TOKEN" "false" "" "9091" "false" "sh" "" "false" "wash")
values=("${DEFAULTS[@]}")

ROW_BASE=5
COL_VAL=28
VAL_W=30
ROW_BTN=$((ROW_BASE + FIELD_COUNT + 2))
ROW_STAT=$((ROW_BTN + 2))

go_ok=false
clean_exit=false

# ─── Terminal helpers ──────────────────────────────
raw_on()  { stty -echo -icanon 2>/dev/null; }
raw_off() { stty echo icanon 2>/dev/null; }
cleanup() { raw_off; printf "${SHOW}${CLR}${HOME1}"; }
trap 'cleanup; exit 1' INT TERM
trap 'if ! $clean_exit; then cleanup; fi; exit 0' EXIT

cursor() { printf "\033[%d;%dH" "$1" "$2"; }
clreol() { printf "\033[0K"; }
clrscr() { printf "${CLR}${HOME1}"; }

# ─── Helpers ───────────────────────────────────────
get_field_type() {
  local i=$1
  if [[ $i -lt $FIELD_COUNT ]]; then
    echo "${FIELDS[$i]%%:*}"
  else
    echo "btn"
  fi
}

get_field_name() {
  local i=$1
  [[ $i -ge $FIELD_COUNT ]] && echo ""
  local rest="${FIELDS[$i]#*:}"
  echo "${rest%%:*}"
}

get_field_label() {
  local i=$1
  local rest="${FIELDS[$i]#*:}"
  echo "${rest#*:}"
}

is_visible() {
  local i=$1
  if [[ $i -ge $FIELD_COUNT ]]; then
    return 0
  fi
  local name; name=$(get_field_name "$i")
  if [[ "$name" == "token_value" && "${values[2]}" != "true" ]]; then
    return 1
  fi
  if [[ "$name" == "service_user" && "${values[8]}" != "true" ]]; then
    return 1
  fi
  return 0
}

visible_indices() {
  local out=()
  for ((i=0; i<TOTAL; i++)); do
    if is_visible "$i"; then
      out+=($i)
    fi
  done
  echo "${out[@]}"
}

# ─── TUI draw ──────────────────────────────────────
draw_static() {
  clrscr
  printf "${HIDE}"
  echo ""
  printf "  ${B}WASH${N} ${D}— Setup & Build${N}"
  echo ""
  echo ""
  printf "  ${D}%s${N}" "Navigation: ↑↓ / Tab · Space: toggle · Enter: edit/confirm"
  echo ""
  echo ""

  for ((i=0; i<FIELD_COUNT; i++)); do
    local row=$((ROW_BASE + i))
    cursor "$row" 0; clreol
    local type; type=$(get_field_type "$i")
    local label; label=$(get_field_label "$i")
    if [[ "$type" == "bool" ]]; then
      printf "  %s  %s" "$CHECK_OFF" "$label"
    else
      printf "  %s:" "$label"
    fi
  done

  # buttons
  cursor "$ROW_BTN" 0; clreol
  printf "      ${D}< Save >${N}           ${D}< Cancel >${N}"

  cursor "$ROW_STAT" 0; clreol
  printf "  ${D}%s${N}" "Ready — choose Save to build, or Cancel to exit"
}

draw_item() {
  local i=$1
  if ! is_visible "$i"; then
    local row=$((ROW_BASE + i))
    cursor "$row" 0; clreol
    return
  fi

  local type; type=$(get_field_type "$i")
  local label; label=$(get_field_label "$i")
  local val="${values[$i]:-}"

  if [[ $i -ge $FIELD_COUNT ]]; then
    return  # buttons drawn separately
  fi

  local row=$((ROW_BASE + i))
  cursor "$row" 0
  if [[ "$type" == "bool" ]]; then
    local chk="$CHECK_OFF"
    [[ "$val" == "true" ]] && chk="$CHECK_ON"
    printf "  ${B}%s${N}  %s" "$chk" "$label"
    clreol
  else
    local display="$val"
    if [[ "$(get_field_name "$i")" == "token_value" && -n "$val" ]]; then
      display="********"
    fi
    printf "  %s: " "$label"
    printf "${B}[${N}%-${VAL_W}s${B}]${N}" "$display"
    clreol
  fi
}

highlight_item() {
  local i=$1
  if ! is_visible "$i"; then return; fi

  local type; type=$(get_field_type "$i")
  local label; label=$(get_field_label "$i")
  local val="${values[$i]:-}"

  if [[ $i -ge $FIELD_COUNT ]]; then
    cursor "$ROW_BTN" 0; clreol
    if [[ $i -eq $BTN_SAVE ]]; then
      printf "      ${B}${G}%s${N}${D}           ${B}%s${N}" "< Save >" "< Cancel >"
    else
      printf "      ${D}${B}%s${N}           ${B}${G}%s${N}" "< Save >" "< Cancel >"
    fi
    return
  fi

  local row=$((ROW_BASE + i))
  cursor "$row" 0
  if [[ "$type" == "bool" ]]; then
    local chk="$CHECK_OFF"
    [[ "$val" == "true" ]] && chk="$CHECK_ON"
    printf "  ${B}${G}%s${N} ${B}${G}%s${N}" "$chk" "$label"
    clreol
  else
    local display="$val"
    if [[ "$(get_field_name "$i")" == "token_value" && -n "$val" ]]; then
      display="********"
    fi
    printf "  ${B}${G}%s${N}${B}${G}: ${N}" "$label"
    printf "${B}${G}[${N}${G}%-${VAL_W}s${N}${B}${G}]${N}" "$display"
    clreol
  fi

  # redraw buttons as dim
  cursor "$ROW_BTN" 0; clreol
  printf "      ${D}< Save >${N}           ${D}< Cancel >${N}"
}

status() {
  local msg="$1"
  local color="${2:-$C}"
  cursor "$ROW_STAT" 0; clreol
  printf "  ${color}%s${N}" "$msg"
}

unhighlight_item() {
  draw_item "$1"
}

# ─── Input loop ────────────────────────────────────
tui_loop() {
  raw_on
  draw_static
  for ((i=0; i<FIELD_COUNT; i++)); do draw_item "$i"; done

  local vis=($(visible_indices))
  local cur="${vis[0]}"
  highlight_item "$cur"
  status "Ready"

  local editing=false
  local edit_old=""

  while true; do
    IFS= read -r -N1 ch || true
    if [[ -z "$ch" ]]; then continue; fi

    local key="$ch"

    # escape sequences
    if [[ "$ch" == $'\033' ]]; then
      IFS= read -r -N1 ch2 || true
      if [[ "$ch2" == "[" || "$ch2" == "O" ]]; then
        IFS= read -r -N1 ch3 || true
        case "$ch3" in
          A) key="UP" ;; B) key="DOWN" ;;
          C) key="RIGHT" ;; D) key="LEFT" ;;
          *) key="" ;;
        esac
      elif [[ "$ch2" == $'\033' ]]; then
        key="ESC"
      else
        key=""
      fi
    fi
    [[ -z "$key" ]] && continue

    # current visible
    local vis=($(visible_indices))
    local vis_len=${#vis[@]}
    local vi=-1
    for ((v=0; v<vis_len; v++)); do
      [[ "${vis[$v]}" -eq "$cur" ]] && { vi=$v; break; }
    done
    [[ $vi -eq -1 ]] && { cur="${vis[0]}"; vi=0; }

    if $editing; then
      # ── Edit mode ─────────────────────────────────
      local fname; fname=$(get_field_name "$cur")
      if [[ "$key" == $'\n' || "$key" == $'\r' ]]; then
        if [[ "$fname" == "port" ]]; then
          local pv="${values[$cur]}"
          if ! [[ "$pv" =~ ^[0-9]+$ ]] || [[ "$pv" -lt 1 ]] || [[ "$pv" -gt 65535 ]]; then
            status "Port must be 1–65535" "$R"
            values[$cur]="$edit_old"
            draw_item "$cur"
            highlight_item "$cur"
            continue
          fi
        fi
        editing=false
        draw_item "$cur"
        local nvi=$((vi + 1))
        [[ $nvi -ge $vis_len ]] && nvi=0
        unhighlight_item "$cur"
        cur="${vis[$nvi]}"
        highlight_item "$cur"
        status "Ready"
      elif [[ "$key" == "ESC" ]]; then
        values[$cur]="$edit_old"
        editing=false
        draw_item "$cur"
        highlight_item "$cur"
        status "Cancelled"
      elif [[ "$key" == $'\x7f' ]]; then
        values[$cur]="${values[$cur]:0:-1}"
        highlight_item "$cur"
      elif [[ "$ch" =~ [[:print:]] ]]; then
        values[$cur]+="$ch"
        highlight_item "$cur"
      fi
    else
      # ── Navigation mode ──────────────────────────
      case "$key" in
        UP)
          unhighlight_item "$cur"
          local pvi=$((vi - 1))
          [[ $pvi -lt 0 ]] && pvi=$((vis_len - 1))
          cur="${vis[$pvi]}"
          highlight_item "$cur"
          ;;
        DOWN|$'\t')
          unhighlight_item "$cur"
          local nvi=$((vi + 1))
          [[ $nvi -ge $vis_len ]] && nvi=0
          cur="${vis[$nvi]}"
          highlight_item "$cur"
          ;;
        ' ')
          local ftype; ftype=$(get_field_type "$cur")
          if [[ "$ftype" == "bool" ]]; then
            [[ "${values[$cur]}" == "true" ]] && values[$cur]="false" || values[$cur]="true"
            draw_item "$cur"
            highlight_item "$cur"
            local fname; fname=$(get_field_name "$cur")
            if [[ "$fname" == "set_token" ]]; then
              if [[ "${values[2]}" == "true" ]]; then
                draw_item 3
              else
                local tv_row=$((ROW_BASE + 3))
                cursor "$tv_row" 0; clreol
              fi
            fi
            if [[ "$fname" == "install_service" ]]; then
              if [[ "${values[8]}" == "true" ]]; then
                draw_item 9
              else
                local su_row=$((ROW_BASE + 9))
                cursor "$su_row" 0; clreol
              fi
            fi
          fi
          ;;
        $'\n'|$'\r')
          local ftype; ftype=$(get_field_type "$cur")
          if [[ $cur -ge $FIELD_COUNT ]]; then
            # button pressed
            if [[ $cur -eq $BTN_SAVE ]]; then
              save_and_build
              return
            else
              clean_exit=true
              cleanup
              echo ""
              echo "  Setup cancelled."
              echo ""
              exit 0
            fi
          elif [[ "$ftype" == "bool" ]]; then
            [[ "${values[$cur]}" == "true" ]] && values[$cur]="false" || values[$cur]="true"
            draw_item "$cur"
            highlight_item "$cur"
            local fname; fname=$(get_field_name "$cur")
            if [[ "$fname" == "set_token" ]]; then
              if [[ "${values[2]}" == "true" ]]; then
                draw_item 3
              else
                local tv_row=$((ROW_BASE + 3))
                cursor "$tv_row" 0; clreol
              fi
            fi
            if [[ "$fname" == "install_service" ]]; then
              if [[ "${values[8]}" == "true" ]]; then
                draw_item 9
              else
                local su_row=$((ROW_BASE + 9))
                cursor "$su_row" 0; clreol
              fi
            fi
          else
            edit_old="${values[$cur]}"
            editing=true
            status "Editing... Enter=confirm, Esc=cancel" "$Y"
            highlight_item "$cur"
          fi
          ;;
        *)
          if [[ "$ch" =~ [[:print:]] ]]; then
            local ftype; ftype=$(get_field_type "$cur")
            if [[ "$ftype" == "str" && $cur -lt $FIELD_COUNT ]]; then
              local fname; fname=$(get_field_name "$cur")
              if [[ "$fname" != "token_value" || "${values[2]}" == "true" ]]; then
                edit_old="${values[$cur]}"
                values[$cur]="$ch"
                editing=true
                status "Editing... Enter=confirm, Esc=cancel" "$Y"
                highlight_item "$cur"
              fi
            fi
          fi
          ;;
      esac
    fi
  done
}

# ─── Go ─────────────────────────────────────────────
check_go() {
  if command -v go &>/dev/null; then go_ok=true; return 0; fi
  if [[ -x /usr/local/go/bin/go ]]; then
    export PATH="/usr/local/go/bin:$PATH"
    go_ok=true; return 0
  fi
  return 1
}

install_go() {
  echo ""
  echo "  ${C}Go not found. Installing latest Go...${N}"
  echo ""

  local arch
  case "$(uname -m)" in
    x86_64)  arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "  ${R}Unsupported architecture: $(uname -m)${N}"; return 1 ;;
  esac

  local go_ver=""
  if command -v curl &>/dev/null; then
    go_ver=$(curl -sL https://go.dev/VERSION?m=text 2>/dev/null | head -1)
  elif command -v wget &>/dev/null; then
    go_ver=$(wget -qO- https://go.dev/VERSION?m=text 2>/dev/null | head -1)
  fi
  [[ -z "$go_ver" ]] && go_ver="go1.24.0"
  go_ver="${go_ver#go}"

  local url="https://go.dev/dl/go${go_ver}.linux-${arch}.tar.gz"
  local tmpf="/tmp/go${go_ver}.tar.gz"

  echo "  ${D}Downloading go${go_ver} for ${arch}...${N}"
  if command -v curl &>/dev/null; then
    curl -# -L "$url" -o "$tmpf"
  elif command -v wget &>/dev/null; then
    wget --show-progress -q "$url" -O "$tmpf"
  else
    echo "  ${R}Error: neither curl nor wget found. Install one first.${N}"
    return 1
  fi

  echo ""
  echo "  ${D}Extracting to /usr/local/go (sudo required)...${N}"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "$tmpf"
  export PATH="/usr/local/go/bin:$PATH"

  if ! grep -q '/usr/local/go/bin' ~/.profile 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
    echo "  ${D}Added /usr/local/go/bin to ~/.profile${N}"
  fi

  rm -f "$tmpf"
  go_ok=true
  echo ""
  echo "  ${G}Go ${go_ver} installed.${N}"
}

# ─── Systemd Service ────────────────────────────────
install_systemd_service() {
  local svc_user="${values[9]}"
  local svc_dir="$PWD"
  local svc_file="/etc/systemd/system/wash.service"

  echo ""
  echo "  ${C}Installing systemd service...${N}"
  echo ""

  # Create user if not exists
  if ! id "$svc_user" &>/dev/null; then
    echo "  ${D}Creating system user '${svc_user}'...${N}"
    sudo useradd -r -s /usr/sbin/nologin -M "$svc_user" 2>/dev/null || true
  fi

  # Set directory ownership
  sudo chown -R "$svc_user":"$svc_user" "$svc_dir"

  # Stop & disable existing service
  if systemctl is-active --quiet wash 2>/dev/null; then
    echo "  ${D}Stopping existing wash service...${N}"
    sudo systemctl stop wash
  fi
  if systemctl is-enabled --quiet wash 2>/dev/null; then
    sudo systemctl disable wash
  fi

  # Write unit file
  local env_file="${svc_dir}/.env"
  local env_line=""
  [[ -f "$env_file" ]] && env_line="EnvironmentFile=${env_file}"

  sudo tee "$svc_file" > /dev/null <<UNIT
[Unit]
Description=WASH (Web Accessible Shell)
Documentation=https://github.com/belov/WAShell
After=network.target

[Service]
Type=simple
User=${svc_user}
Group=${svc_user}
WorkingDirectory=${svc_dir}
ExecStart=${svc_dir}/WASH
Restart=on-failure
RestartSec=5
${env_line}
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
PrivateTmp=true
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
UNIT

  sudo systemctl daemon-reload
  sudo systemctl enable wash
  sudo systemctl start wash

  echo "  ${G}✓${N} systemd service installed and started"
  echo "  ${D}Manage: systemctl {status|start|stop|restart} wash${N}"
}

# ─── Save & Build ─────────────────────────────────
save_and_build() {
  raw_off
  printf "${SHOW}${CLR}${HOME1}"

  echo ""
  echo "  ${B}${C}── Saving configuration ──${N}"
  echo ""

  # Validate port
  local port_val="${values[4]}"
  if ! [[ "$port_val" =~ ^[0-9]+$ ]] || [[ "$port_val" -lt 1 ]] || [[ "$port_val" -gt 65535 ]]; then
    echo "  ${R}Error: Port must be 1–65535${N}"
    echo ""
    echo "  ${D}Press Enter to go back...${N}"
    read -r
    raw_on
    draw_static
    for ((i=0; i<FIELD_COUNT; i++)); do draw_item "$i"; done
    local __vis=($(visible_indices))
    local __cur="${__vis[0]}"
    highlight_item "$__cur"
    status "Ready"
    return
  fi

  # config.yaml
  cat > config.yaml <<'YAML'
# WASH (Web Accessible Shell) configuration file
# Generated by setup-linux.sh

# Enable OS authentication (true/false)
os_auth: __OS_AUTH__

# Environment variable name for the token
token: __TOKEN_ENV__

# Port on which the application will run
port: __PORT__

# Listen on 0.0.0.0 (true) or 127.0.0.1 (false)
allow_0: __ALLOW_0__

# Working directory (empty = user home)
work_dir: __WORK_DIR__

# Shell command for interactive sessions (e.g., bash, zsh, sh)
shell: __SHELL__
YAML

  sed -i "s/__OS_AUTH__/${values[0]}/g" config.yaml
  sed -i "s|__TOKEN_ENV__|${values[1]}|g" config.yaml
  sed -i "s/__PORT__/${values[4]}/g" config.yaml
  sed -i "s/__ALLOW_0__/${values[5]}/g" config.yaml
  sed -i "s|__WORK_DIR__|${values[7]}|g" config.yaml
  sed -i "s|__SHELL__|${values[6]}|g" config.yaml

  echo "  ${G}✓${N} config.yaml written"

  # .env
  if [[ "${values[2]}" == "true" && -n "${values[3]}" ]]; then
    cat > .env <<ENV
# WASH environment variables
# Generated by setup-linux.sh

${values[1]}=${values[3]}
ENV
    chmod 600 .env
    echo "  ${G}✓${N} .env written with token"
  fi

  echo ""
  echo "  ${B}${C}── Building WASH ──${N}"
  echo ""

  if ! $go_ok; then
    if ! check_go; then
      echo "  ${R}Error: Go is not installed.${N}"
      echo "  Run the script again and allow Go installation."
      echo ""
      read -rp "  Press Enter to go back... "
      raw_on
      draw_static
      for ((i=0; i<FIELD_COUNT; i++)); do draw_item "$i"; done
      local __vis=($(visible_indices))
      local __cur="${__vis[0]}"
      highlight_item "$__cur"
      status "Ready"
      return
    fi
  fi

  go build -o WASH .
  echo ""
  echo "  ${G}${B}✓ Build successful!${N}  Binary: ${B}WASH${N}"

  if [[ "${values[8]}" == "true" ]]; then
    install_systemd_service
  fi

  echo ""
  echo "  ${D}Run:${N}"
  echo "    ${B}./WASH -token=YOUR_TOKEN -port=9091${N}"
  echo "    ${B}./WASH -os-auth -port=9091${N}"
  echo ""
  echo "  ${D}Edit config.yaml for permanent settings.${N}"
  echo ""
  read -rp "  Press Enter to exit... "

  clean_exit=true
  cleanup
  echo ""
  exit 0
}

# ─── Main ──────────────────────────────────────────
printf "${CLR}${HOME1}${SHOW}"
echo ""
echo "  ${B}WASH — Setup & Build${N}"
echo "  ${D}━━━━━━━━━━━━━━━━━━━━━${N}"
echo ""

if ! check_go; then
  install_go
else
  echo "  ${G}✓${N} Go $(go version | grep -oP 'go\S+' | tr -d go) found"
fi
echo ""

echo "  ${D}Starting configuration TUI...${N}"
sleep 0.5

tui_loop
