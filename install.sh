#!/usr/bin/env bash
# =============================================================================
# MTPanel Install Script
# Usage:
#   curl -fsSL https://get.example.com/install.sh | bash
#   curl -fsSL https://get.example.com/install.sh | bash -s -- --port 8080
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Colour helpers
# ---------------------------------------------------------------------------
if [ -t 1 ] && command -v tput &>/dev/null && tput colors &>/dev/null; then
  RED=$(tput setaf 1); GREEN=$(tput setaf 2); YELLOW=$(tput setaf 3)
  CYAN=$(tput setaf 6); BOLD=$(tput bold); RESET=$(tput sgr0)
else
  RED=''; GREEN=''; YELLOW=''; CYAN=''; BOLD=''; RESET=''
fi

info()    { printf "%s[INFO]%s  %s\n"    "${CYAN}"   "${RESET}" "$*"; }
success() { printf "%s[OK]%s    %s\n"    "${GREEN}"  "${RESET}" "$*"; }
warn()    { printf "%s[WARN]%s  %s\n"    "${YELLOW}" "${RESET}" "$*" >&2; }
die()     { printf "%s[ERROR]%s %s\n"    "${RED}"    "${RESET}" "$*" >&2; exit 1; }
step()    { printf "\n%s==> %s%s\n"      "${BOLD}"   "$*" "${RESET}"; }

# ---------------------------------------------------------------------------
# Defaults (overridable via flags)
# ---------------------------------------------------------------------------
PANEL_PORT=8080
MTPROXY_PORT=443
GITHUB_REPO="NikitaKHS/mtpanel"
INSTALL_DIR="/opt/mtpanel"
MTPROXY_DIR="/opt/mtproxy"
DATA_DIR="/var/lib/mtpanel"
CONFIG_DIR="/etc/mtpanel"
PANEL_USER="mtpanel"
CONFIG_FILE="${CONFIG_DIR}/config.json"
SERVICE_NAME="mtpanel"
BINARY_NAME="mtpanel"
DOWNLOADED_WEB_DIST=""

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --port)        PANEL_PORT="${2:?'--port requires a value'}"; shift 2 ;;
      --mtproxy-port) MTPROXY_PORT="${2:?'--mtproxy-port requires a value'}"; shift 2 ;;
      --repo)        GITHUB_REPO="${2:?'--repo requires a value'}"; shift 2 ;;
      --help|-h)
        cat <<EOF
MTPanel Installer

Options:
  --port <port>          Panel listen port (default: 8080)
  --mtproxy-port <port>  MTProxy listen port (default: 443)
  --repo <owner/repo>    GitHub repo for releases (default: NikitaKHS/mtpanel)
  --help                 Show this help
EOF
        exit 0
        ;;
      *) die "Unknown argument: $1. Run with --help for usage." ;;
    esac
  done
}

# ---------------------------------------------------------------------------
# Root / sudo check
# ---------------------------------------------------------------------------
require_root() {
  if [[ $EUID -ne 0 ]]; then
    die "This script must be run as root. Try: sudo bash install.sh"
  fi
}

# ---------------------------------------------------------------------------
# OS detection
# ---------------------------------------------------------------------------
detect_os() {
  step "Detecting operating system"

  OS_ID=""
  OS_FAMILY=""   # debian | rhel | arch
  PKG_MANAGER=""
  PKG_INSTALL=""

  if [[ -f /etc/os-release ]]; then
    # shellcheck source=/dev/null
    source /etc/os-release
    OS_ID="${ID:-unknown}"
    OS_ID_LIKE="${ID_LIKE:-}"
  else
    die "/etc/os-release not found. Cannot detect OS."
  fi

  case "${OS_ID}" in
    ubuntu|debian|raspbian)
      OS_FAMILY="debian"
      PKG_MANAGER="apt-get"
      PKG_INSTALL="apt-get install -y -q"
      ;;
    centos|rhel|fedora|rocky|almalinux|ol)
      OS_FAMILY="rhel"
      if command -v dnf &>/dev/null; then
        PKG_MANAGER="dnf"; PKG_INSTALL="dnf install -y -q"
      else
        PKG_MANAGER="yum"; PKG_INSTALL="yum install -y -q"
      fi
      ;;
    arch|manjaro|endeavouros)
      OS_FAMILY="arch"
      PKG_MANAGER="pacman"
      PKG_INSTALL="pacman -Sy --noconfirm --needed"
      ;;
    *)
      # Fallback via ID_LIKE
      if echo "${OS_ID_LIKE}" | grep -qi "debian"; then
        OS_FAMILY="debian"; PKG_MANAGER="apt-get"; PKG_INSTALL="apt-get install -y -q"
      elif echo "${OS_ID_LIKE}" | grep -qi "rhel\|fedora"; then
        OS_FAMILY="rhel"
        if command -v dnf &>/dev/null; then
          PKG_MANAGER="dnf"; PKG_INSTALL="dnf install -y -q"
        else
          PKG_MANAGER="yum"; PKG_INSTALL="yum install -y -q"
        fi
      elif echo "${OS_ID_LIKE}" | grep -qi "arch"; then
        OS_FAMILY="arch"; PKG_MANAGER="pacman"; PKG_INSTALL="pacman -Sy --noconfirm --needed"
      else
        die "Unsupported OS: ${OS_ID}. Supported: Ubuntu, Debian, CentOS, RHEL, Fedora, Rocky, Alma, Arch."
      fi
      ;;
  esac

  info "Detected OS: ${OS_ID} (family: ${OS_FAMILY})"
}

# ---------------------------------------------------------------------------
# Architecture detection
# ---------------------------------------------------------------------------
detect_arch() {
  step "Detecting system architecture"

  RAW_ARCH=$(uname -m)
  case "${RAW_ARCH}" in
    x86_64|amd64)    ARCH="amd64" ;;
    aarch64|arm64)   ARCH="arm64" ;;
    armv7l|armv6l)   die "32-bit ARM is not supported. Please use a 64-bit OS." ;;
    *)               die "Unsupported architecture: ${RAW_ARCH}" ;;
  esac

  info "Architecture: ${ARCH}"
}

# ---------------------------------------------------------------------------
# Dependency checks
# ---------------------------------------------------------------------------
check_deps() {
  step "Checking dependencies"

  # systemctl is mandatory
  if ! command -v systemctl &>/dev/null; then
    die "systemd is required but systemctl was not found. MTPanel only supports systemd-based systems."
  fi

  # Downloader: prefer curl, fall back to wget
  DOWNLOADER=""
  if command -v curl &>/dev/null; then
    DOWNLOADER="curl"
    info "Using curl for downloads"
  elif command -v wget &>/dev/null; then
    DOWNLOADER="wget"
    info "Using wget for downloads"
  else
    warn "Neither curl nor wget found. Attempting to install curl..."
    install_pkg curl
    DOWNLOADER="curl"
  fi

  # jq for parsing GitHub API (optional, we have a fallback)
  if command -v jq &>/dev/null; then
    HAS_JQ=true
  else
    HAS_JQ=false
    warn "jq not found - will use grep/sed to parse GitHub API response"
  fi

  success "Dependency check passed"
}

# ---------------------------------------------------------------------------
# Package installation helper
# ---------------------------------------------------------------------------
install_pkg() {
  local pkg="$1"
  info "Installing ${pkg}..."
  case "${OS_FAMILY}" in
    debian) apt-get update -qq && ${PKG_INSTALL} "${pkg}" ;;
    rhel)   ${PKG_INSTALL} "${pkg}" ;;
    arch)   ${PKG_INSTALL} "${pkg}" ;;
  esac
}

# ---------------------------------------------------------------------------
# Download helper (curl or wget)
# ---------------------------------------------------------------------------
download() {
  local url="$1" dest="$2"
  if [[ "${DOWNLOADER}" == "curl" ]]; then
    curl -fsSL --retry 3 --retry-delay 2 -o "${dest}" "${url}"
  else
    wget -q --tries=3 -O "${dest}" "${url}"
  fi
}

download_stdout() {
  local url="$1"
  if [[ "${DOWNLOADER}" == "curl" ]]; then
    curl -fsSL --retry 3 --retry-delay 2 "${url}"
  else
    wget -q --tries=3 -O - "${url}"
  fi
}

# ---------------------------------------------------------------------------
# Fetch latest release tag from GitHub
# ---------------------------------------------------------------------------
get_latest_release() {
  local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
  local response tag

  if ! response=$(download_stdout "${api_url}" 2>/dev/null); then
    return 1
  fi

  if [[ "${HAS_JQ}" == "true" ]]; then
    tag=$(echo "${response}" | jq -r '.tag_name') || return 1
  else
    tag=$(echo "${response}" | grep -o '"tag_name": *"[^"]*"' | head -1 | \
          sed 's/.*"tag_name": *"\([^"]*\)".*/\1/') || return 1
  fi

  [[ -z "${tag}" || "${tag}" == "null" ]] && return 1

  echo "${tag}"
}

# ---------------------------------------------------------------------------
# Download and verify panel binary
# ---------------------------------------------------------------------------
ensure_frontend_tools() {
  local pkgs=()

  if ! command -v git &>/dev/null; then
    pkgs+=("git")
  fi
  if ! command -v node &>/dev/null; then
    case "${OS_FAMILY}" in
      debian) pkgs+=("nodejs") ;;
      rhel)   pkgs+=("nodejs") ;;
      arch)   pkgs+=("nodejs") ;;
    esac
  fi
  if ! command -v npm &>/dev/null; then
    case "${OS_FAMILY}" in
      debian) pkgs+=("npm") ;;
      rhel)   pkgs+=("npm") ;;
      arch)   pkgs+=("npm") ;;
    esac
  fi

  if [[ ${#pkgs[@]} -gt 0 ]]; then
    step "Installing build tools (fallback mode)"
    for p in "${pkgs[@]}"; do
      install_pkg "${p}"
    done
  fi

  command -v git &>/dev/null || die "Git is required for source-build fallback but not available"
  command -v npm &>/dev/null || die "npm is required for source-build fallback but not available"
}

ensure_build_tools() {
  ensure_frontend_tools

  if ! command -v go &>/dev/null; then
    case "${OS_FAMILY}" in
      debian) install_pkg "golang-go" ;;
      rhel)   install_pkg "golang" ;;
      arch)   install_pkg "go" ;;
    esac
  fi

  command -v go &>/dev/null || die "Go compiler is required for source-build fallback but not available"
}

build_frontend_for_tag() {
  local tag="$1"
  local workdir="/tmp/mtpanel-web-src.$$"
  local repo_url="https://github.com/${GITHUB_REPO}.git"
  local tmp_web="/tmp/mtpanel-web-dist.$$"

  ensure_frontend_tools

  rm -rf "${workdir}"
  if ! git clone --depth 1 --branch "${tag}" "${repo_url}" "${workdir}" >/dev/null 2>&1; then
    warn "Failed to clone tag ${tag} for frontend build fallback"
    rm -rf "${workdir}"
    return 1
  fi

  info "Building frontend bundle from source tag ${tag}"
  (
    cd "${workdir}/frontend" && \
    npm ci --silent && \
    npm run build >/dev/null
  ) || {
    rm -rf "${workdir}"
    return 1
  }

  if [[ ! -d "${workdir}/web/dist" ]]; then
    rm -rf "${workdir}"
    return 1
  fi
  rm -rf "${tmp_web}"
  mkdir -p "${tmp_web}"
  cp -r "${workdir}/web/dist/." "${tmp_web}/"
  rm -rf "${workdir}"

  DOWNLOADED_WEB_DIST="${tmp_web}"
  success "Frontend bundle prepared"
  return 0
}

build_binary_from_source() {
  step "Building MTPanel from source (release fallback)"

  ensure_build_tools

  local workdir="/tmp/mtpanel-src.$$"
  local repo_url="https://github.com/${GITHUB_REPO}.git"
  local tmp_bin="/tmp/mtpanel-download"
  local tmp_web="/tmp/mtpanel-web-dist.$$"

  rm -rf "${workdir}"
  git clone --depth 1 "${repo_url}" "${workdir}" >/dev/null 2>&1 || \
    die "Failed to clone source repository: ${repo_url}"

  info "Compiling MTPanel with Go ($(go version))"
  (
    cd "${workdir}" && \
    CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" go build -trimpath -ldflags "-s -w" -o "${tmp_bin}" ./cmd/mtpanel
  ) || die "Source build failed"

  info "Building frontend bundle from source"
  (
    cd "${workdir}/frontend" && \
    npm ci --silent && \
    npm run build >/dev/null
  ) || die "Frontend build failed"

  if [[ ! -d "${workdir}/web/dist" ]]; then
    die "Frontend build finished but web/dist not found"
  fi
  rm -rf "${tmp_web}"
  mkdir -p "${tmp_web}"
  cp -r "${workdir}/web/dist/." "${tmp_web}/"

  chmod +x "${tmp_bin}"
  DOWNLOADED_BINARY="${tmp_bin}"
  DOWNLOADED_WEB_DIST="${tmp_web}"
  rm -rf "${workdir}"
  success "Source build completed"
}

download_binary() {
  step "Downloading MTPanel binary"

  RELEASE_TAG=""
  if RELEASE_TAG=$(get_latest_release); then
    info "Latest release: ${RELEASE_TAG}"

    local binary_name="mtpanel-linux-${ARCH}"
    local checksum_name="mtpanel-linux-${ARCH}.sha256"
    local web_bundle_name="web-dist.tar.gz"
    local base_url="https://github.com/${GITHUB_REPO}/releases/download/${RELEASE_TAG}"
    local tmp_bin="/tmp/mtpanel-download"
    local tmp_sum="/tmp/mtpanel-download.sha256"
    local tmp_web_tgz="/tmp/mtpanel-web-dist.tar.gz.$$"
    local tmp_web_dir="/tmp/mtpanel-web-dist.$$"

    info "Downloading binary: ${binary_name}"
    if ! download "${base_url}/${binary_name}" "${tmp_bin}" 2>/dev/null; then
      warn "Release binary not found for ${GITHUB_REPO}/${RELEASE_TAG} (${ARCH})"
      build_binary_from_source
      return
    fi

    # Verify checksum if available
    if download "${base_url}/${checksum_name}" "${tmp_sum}" 2>/dev/null; then
      info "Verifying checksum..."
      # The checksum file may contain just the hash or "hash  filename"
      local expected actual
      expected=$(awk '{print $1}' "${tmp_sum}")
      actual=$(sha256sum "${tmp_bin}" | awk '{print $1}')
      if [[ "${expected}" != "${actual}" ]]; then
        rm -f "${tmp_bin}" "${tmp_sum}"
        die "Checksum mismatch! Expected: ${expected}, got: ${actual}"
      fi
      success "Checksum verified"
      rm -f "${tmp_sum}"
    else
      warn "No checksum file found - skipping verification"
    fi

    chmod +x "${tmp_bin}"
    DOWNLOADED_BINARY="${tmp_bin}"

    info "Fetching frontend assets bundle: ${web_bundle_name}"
    if download "${base_url}/${web_bundle_name}" "${tmp_web_tgz}" 2>/dev/null; then
      rm -rf "${tmp_web_dir}"
      mkdir -p "${tmp_web_dir}"
      if tar -xzf "${tmp_web_tgz}" -C "${tmp_web_dir}" >/dev/null 2>&1; then
        DOWNLOADED_WEB_DIST="${tmp_web_dir}"
        success "Frontend assets bundle downloaded"
      else
        warn "Failed to extract ${web_bundle_name}; trying source-build fallback for frontend"
      fi
      rm -f "${tmp_web_tgz}"
    else
      warn "Release frontend bundle not found; trying source-build fallback for frontend"
    fi

    if [[ -z "${DOWNLOADED_WEB_DIST:-}" || ! -d "${DOWNLOADED_WEB_DIST}" ]]; then
      if ! build_frontend_for_tag "${RELEASE_TAG}"; then
        warn "Frontend fallback build failed; proceeding without static assets"
      fi
    fi

    return
  fi

  warn "No GitHub release found for ${GITHUB_REPO}; switching to source-build fallback"
  build_binary_from_source
}

# ---------------------------------------------------------------------------
# Create system user (idempotent)
# ---------------------------------------------------------------------------
create_user() {
  local username="$1"
  if id "${username}" &>/dev/null; then
    info "User '${username}' already exists - skipping"
  else
    info "Creating system user: ${username}"
    useradd --system --no-create-home --shell /sbin/nologin \
            --comment "MTPanel service account" "${username}"
    success "User '${username}' created"
  fi
}

# ---------------------------------------------------------------------------
# Create directory (idempotent, set owner)
# ---------------------------------------------------------------------------
ensure_dir() {
  local dir="$1" owner="${2:-root}" mode="${3:-755}"
  if [[ -d "${dir}" ]]; then
    info "Directory exists: ${dir}"
  else
    info "Creating directory: ${dir}"
    mkdir -p "${dir}"
  fi
  chown "${owner}:${owner}" "${dir}"
  chmod "${mode}" "${dir}"
}

# ---------------------------------------------------------------------------
# Generate secure random string
# ---------------------------------------------------------------------------
random_string() {
  local len="${1:-32}"
  # Try multiple sources of randomness
  if command -v openssl &>/dev/null; then
    openssl rand -hex "${len}"
  else
    tr -dc 'a-zA-Z0-9' < /dev/urandom | head -c "${len}"
  fi
}

# ---------------------------------------------------------------------------
# Write config file
# ---------------------------------------------------------------------------
write_config() {
  step "Writing configuration"

  # Generate secrets only if not already present in an existing config
  if [[ -f "${CONFIG_FILE}" ]]; then
    info "Existing config found at ${CONFIG_FILE} - preserving secrets"
    # Extract existing secrets to avoid clobbering them
    if command -v jq &>/dev/null; then
      EXISTING_JWT=$(jq -r '.jwt_secret // empty' "${CONFIG_FILE}" 2>/dev/null || echo "")
      EXISTING_PASS=$(jq -r '.admin_password_hash // empty' "${CONFIG_FILE}" 2>/dev/null || echo "")
      EXISTING_SECRET=$(jq -r '.mtproxy_secret // empty' "${CONFIG_FILE}" 2>/dev/null || echo "")
    else
      EXISTING_JWT=$(grep -o '"jwt_secret"[[:space:]]*:[[:space:]]*"[^"]*"' "${CONFIG_FILE}" 2>/dev/null | \
                     sed 's/.*"\([^"]*\)"$/\1/' || echo "")
      EXISTING_PASS=""
      EXISTING_SECRET=""
    fi
    JWT_SECRET="${EXISTING_JWT:-$(random_string 32)}"
    MTPROXY_SECRET="${EXISTING_SECRET:-$(random_string 16)}"
    # Never regenerate password hash - it might have been changed by user
    INITIAL_PASSWORD=""
  else
    info "Generating fresh secrets"
    JWT_SECRET=$(random_string 32)
    MTPROXY_SECRET=$(random_string 16)
    # Generate a random initial password (plain - panel will hash on first run)
    INITIAL_PASSWORD=$(random_string 12)
  fi

  cat > "${CONFIG_FILE}" <<EOF
{
  "listen_addr": ":${PANEL_PORT}",
  "data_dir": "${DATA_DIR}",
  "db_path": "${DATA_DIR}/mtpanel.db",
  "mtproxy_bin_path": "/opt/mtproxy/mtproto-proxy",
  "mtproxy_port": ${MTPROXY_PORT},
  "mtproxy_secret": "${MTPROXY_SECRET}",
  "jwt_secret": "${JWT_SECRET}",
  "jwt_expire_hours": 24,
  "log_level": "info",
  "is_first_run": true
}
EOF

  # Store plain initial password separately for display - panel reads and deletes it
  if [[ -n "${INITIAL_PASSWORD:-}" ]]; then
    echo "${INITIAL_PASSWORD}" > "${CONFIG_DIR}/.initial_password"
    chmod 600 "${CONFIG_DIR}/.initial_password"
  fi

  chown root:${PANEL_USER} "${CONFIG_FILE}"
  chmod 640 "${CONFIG_FILE}"
  success "Config written to ${CONFIG_FILE}"
}

# ---------------------------------------------------------------------------
# Install environment file for systemd
# ---------------------------------------------------------------------------
write_env_file() {
  local env_file="${CONFIG_DIR}/mtpanel.env"

  # Do not overwrite if it already exists (it may contain user customisation)
  if [[ -f "${env_file}" ]]; then
    info "Environment file already exists: ${env_file} - skipping"
    return
  fi

  cat > "${env_file}" <<EOF
# MTPanel environment variables
# Uncomment and set to override config file values
MTPANEL_CONFIG=${CONFIG_FILE}
# MTPANEL_LISTEN=:${PANEL_PORT}
# MTPANEL_LOG_LEVEL=info
EOF

  chown root:${PANEL_USER} "${env_file}"
  chmod 640 "${env_file}"
  success "Environment file written: ${env_file}"
}

# ---------------------------------------------------------------------------
# Install frontend static assets
# ---------------------------------------------------------------------------
install_frontend_assets() {
  local target_dir="${INSTALL_DIR}/web/dist"

  if [[ -z "${DOWNLOADED_WEB_DIST:-}" || ! -d "${DOWNLOADED_WEB_DIST}" ]]; then
    warn "Frontend assets bundle is not available - panel UI may return 404"
    return
  fi

  step "Installing frontend assets"
  rm -rf "${INSTALL_DIR}/web"
  mkdir -p "${target_dir}"
  cp -r "${DOWNLOADED_WEB_DIST}/." "${target_dir}/"
  chown -R root:root "${INSTALL_DIR}/web"
  chmod -R a+rX "${INSTALL_DIR}/web"
  success "Frontend assets installed to ${target_dir}"
}

# ---------------------------------------------------------------------------
# Install panel systemd unit
# ---------------------------------------------------------------------------
install_panel_service() {
  step "Installing MTPanel systemd service"

  local unit_file="/etc/systemd/system/mtpanel.service"

  cat > "${unit_file}" <<EOF
[Unit]
Description=MTPanel - MTProxy Management Panel
Documentation=https://github.com/${GITHUB_REPO}
After=network-online.target
Wants=network-online.target
# Restart if MTProxy service is cycled
PartOf=mtproxy.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=-${CONFIG_DIR}/mtpanel.env
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
ExecReload=/bin/kill -HUP \$MAINPID

Restart=on-failure
RestartSec=5s
TimeoutStartSec=30s
TimeoutStopSec=30s

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mtpanel

# ---- Security hardening ----
NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${DATA_DIR} ${CONFIG_DIR} ${MTPROXY_DIR} /etc/systemd/system
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictRealtime=true
LockPersonality=true
MemoryDenyWriteExecute=false
RestrictNamespaces=true
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM

# Allow systemd management for MTProxy
# The panel calls systemctl to manage mtproxy.service
AmbientCapabilities=
CapabilityBoundingSet=
# Needed to bind ports < 1024 only if panel port is privileged
# CapabilityBoundingSet=CAP_NET_BIND_SERVICE
# AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

  chmod 644 "${unit_file}"
  success "Systemd unit installed: ${unit_file}"
}

# ---------------------------------------------------------------------------
# Enable and start panel service
# ---------------------------------------------------------------------------
start_panel_service() {
  step "Enabling and starting MTPanel service"

  systemctl daemon-reload

  if systemctl is-enabled "${SERVICE_NAME}" &>/dev/null; then
    info "Service already enabled - skipping enable"
  else
    systemctl enable "${SERVICE_NAME}"
    success "Service enabled"
  fi

  if systemctl is-active "${SERVICE_NAME}" &>/dev/null; then
    info "Service is running - restarting to apply new binary"
    systemctl restart "${SERVICE_NAME}"
  else
    systemctl start "${SERVICE_NAME}"
  fi

  # Wait for the service to come up
  local attempts=0
  local max_attempts=12
  while ! systemctl is-active --quiet "${SERVICE_NAME}"; do
    attempts=$((attempts + 1))
    if [[ ${attempts} -ge ${max_attempts} ]]; then
      warn "Service did not start within 60 seconds."
      warn "Check logs with: journalctl -u ${SERVICE_NAME} -n 50"
      return 1
    fi
    info "Waiting for service... (${attempts}/${max_attempts})"
    sleep 5
  done

  success "MTPanel service is running"
}

# ---------------------------------------------------------------------------
# Firewall hints
# ---------------------------------------------------------------------------
firewall_hints() {
  step "Firewall configuration"

  local fw_detected=false

  if command -v ufw &>/dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
    fw_detected=true
    info "UFW detected. Run these commands to open ports:"
    printf "    ${CYAN}ufw allow %s/tcp   # Panel UI${RESET}\n" "${PANEL_PORT}"
    printf "    ${CYAN}ufw allow %s/tcp   # MTProxy${RESET}\n"  "${MTPROXY_PORT}"
    printf "    ${CYAN}ufw reload${RESET}\n"
  fi

  if command -v firewall-cmd &>/dev/null && firewall-cmd --state 2>/dev/null | grep -q "running"; then
    fw_detected=true
    info "firewalld detected. Run these commands:"
    printf "    ${CYAN}firewall-cmd --permanent --add-port=%s/tcp${RESET}\n" "${PANEL_PORT}"
    printf "    ${CYAN}firewall-cmd --permanent --add-port=%s/tcp${RESET}\n" "${MTPROXY_PORT}"
    printf "    ${CYAN}firewall-cmd --reload${RESET}\n"
  fi

  if ! ${fw_detected}; then
    # Check raw iptables rules
    if command -v iptables &>/dev/null; then
      info "iptables detected (no frontend). Suggested rules:"
      printf "    ${CYAN}iptables -A INPUT -p tcp --dport %s -j ACCEPT${RESET}\n" "${PANEL_PORT}"
      printf "    ${CYAN}iptables -A INPUT -p tcp --dport %s -j ACCEPT${RESET}\n" "${MTPROXY_PORT}"
      printf "    ${CYAN}iptables-save > /etc/iptables/rules.v4${RESET}\n"
    else
      info "No firewall detected - ports should be accessible already"
    fi
  fi
}

# ---------------------------------------------------------------------------
# Detect server IP
# ---------------------------------------------------------------------------
get_server_ip() {
  local ip=""
  if command -v curl &>/dev/null; then
    ip=$(curl -fsSL --max-time 3 https://api.ipify.org 2>/dev/null || true)
  fi
  if [[ -z "${ip}" ]]; then
    ip=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "YOUR_SERVER_IP")
  fi
  echo "${ip}"
}

# ---------------------------------------------------------------------------
# Print success summary
# ---------------------------------------------------------------------------
print_summary() {
  local server_ip
  server_ip=$(get_server_ip)

  local initial_password="(preserved from existing install)"
  if [[ -f "${CONFIG_DIR}/.initial_password" ]]; then
    initial_password=$(cat "${CONFIG_DIR}/.initial_password")
  fi

  echo ""
  printf "%s%s%s\n" "${BOLD}${GREEN}" "--------------------------------------------------------" "${RESET}"
  printf "%s%s%s\n" "${BOLD}${GREEN}" "          MTPanel installed successfully!               " "${RESET}"
  printf "%s%s%s\n" "${BOLD}${GREEN}" "--------------------------------------------------------" "${RESET}"
  echo ""
  printf "  %sPanel URL:%s       http://%s:%s\n"    "${BOLD}" "${RESET}" "${server_ip}" "${PANEL_PORT}"
  printf "  %sInitial password:%s %s\n"             "${BOLD}" "${RESET}" "${initial_password}"
  printf "  %sConfig file:%s     %s\n"              "${BOLD}" "${RESET}" "${CONFIG_FILE}"
  printf "  %sData directory:%s  %s\n"              "${BOLD}" "${RESET}" "${DATA_DIR}"
  printf "  %sService logs:%s    journalctl -u mtpanel -f\n" "${BOLD}" "${RESET}"
  echo ""
  printf "  %sChange your password immediately after first login!%s\n" "${YELLOW}" "${RESET}"
  echo ""
  printf "  %sUseful commands:%s\n" "${BOLD}" "${RESET}"
  printf "    systemctl status mtpanel\n"
  printf "    systemctl restart mtpanel\n"
  printf "    journalctl -u mtpanel -n 100 --no-pager\n"
  echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  echo ""
  printf "%s%s%s\n" "${BOLD}${CYAN}" \
    "  MTPanel Installer - Self-Hosted MTProxy Management" "${RESET}"
  printf "%s%s%s\n" "${CYAN}" \
    "  https://github.com/${GITHUB_REPO}" "${RESET}"
  echo ""
  info "Прогресс может идти неравномерно, установка продолжается."
  info "Progress can be non-linear; installation is still in progress."

  parse_args "$@"
  require_root
  detect_os
  detect_arch
  check_deps

  step "Preparing directories and users"
  create_user "${PANEL_USER}"
  ensure_dir "${INSTALL_DIR}"          root   755
  ensure_dir "${MTPROXY_DIR}"          root   755
  ensure_dir "${DATA_DIR}"             "${PANEL_USER}" 750
  ensure_dir "${CONFIG_DIR}"           root   755

  download_binary

  step "Installing binary"
  # Backup existing binary if present (idempotent upgrade support)
  if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
    cp "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}.bak"
    info "Previous binary backed up to ${INSTALL_DIR}/${BINARY_NAME}.bak"
  fi
  mv "${DOWNLOADED_BINARY}" "${INSTALL_DIR}/${BINARY_NAME}"
  chown root:root "${INSTALL_DIR}/${BINARY_NAME}"
  chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
  success "Binary installed: ${INSTALL_DIR}/${BINARY_NAME}"
  install_frontend_assets

  write_config
  write_env_file
  install_panel_service

  if ! start_panel_service; then
    die "MTPanel service failed to start. Run: journalctl -u mtpanel -n 50"
  fi

  firewall_hints
  print_summary
}

main "$@"
