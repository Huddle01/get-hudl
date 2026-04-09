#!/bin/sh
set -e

# ─── Config ──────────────────────────────────────────────────────────────────
REPO="Huddle01/hudl"
BINARY="hudl"
INSTALL_DIR="/usr/local/bin"

# ─── Colors ──────────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  BOLD=$(printf '\033[1m')
  DIM=$(printf '\033[2m')
  RESET=$(printf '\033[0m')
  CYAN=$(printf '\033[36m')
  GREEN=$(printf '\033[32m')
  RED=$(printf '\033[31m')
  YELLOW=$(printf '\033[33m')
  WHITE=$(printf '\033[97m')
else
  BOLD="" DIM="" RESET="" CYAN="" GREEN="" RED="" YELLOW="" WHITE=""
fi

# ─── Helpers ─────────────────────────────────────────────────────────────────
info()    { printf "  ${CYAN}${BOLD}info${RESET}  %s\n" "$1"; }
success() { printf "  ${GREEN}${BOLD}  ok${RESET}  %s\n" "$1"; }
warn()    { printf "  ${YELLOW}${BOLD}warn${RESET}  %s\n" "$1"; }
fail()    { printf "  ${RED}${BOLD}fail${RESET}  %s\n" "$1"; exit 1; }

# ─── Banner ──────────────────────────────────────────────────────────────────
banner() {
  printf "\n"
  printf "  ${CYAN}            ██████████████████${RESET}\n"
  printf "  ${CYAN}        ████${RESET}                    ${CYAN}████${RESET}\n"
  printf "  ${CYAN}      ██${RESET}  ${CYAN}████${RESET}    ${CYAN}████████${RESET}    ${CYAN}████${RESET}  ${CYAN}██${RESET}\n"
  printf "  ${CYAN}    ██${RESET}  ${CYAN}████${RESET}    ${CYAN}██████████${RESET}    ${CYAN}████${RESET}  ${CYAN}██${RESET}\n"
  printf "  ${CYAN}  ██${RESET}  ${CYAN}██████${RESET}                    ${CYAN}██████${RESET}  ${CYAN}██${RESET}\n"
  printf "  ${CYAN}  ██${RESET}  ${CYAN}██████${RESET}                    ${CYAN}██████${RESET}  ${CYAN}██${RESET}\n"
  printf "  ${CYAN}    ██${RESET}  ${CYAN}████${RESET}    ${CYAN}██████████${RESET}    ${CYAN}████${RESET}  ${CYAN}██${RESET}\n"
  printf "  ${CYAN}      ██${RESET}  ${CYAN}████${RESET}    ${CYAN}████████${RESET}    ${CYAN}████${RESET}  ${CYAN}██${RESET}\n"
  printf "  ${CYAN}        ████${RESET}                    ${CYAN}████${RESET}\n"
  printf "  ${CYAN}            ██████████████████${RESET}\n"
  printf "\n"
  printf "  ${BOLD}${WHITE}            h u d d l e 0 1${RESET}\n"
  printf "  ${DIM}         The Future of Compute${RESET}\n"
  printf "\n"
}

# ─── Detect Platform ─────────────────────────────────────────────────────────
detect_platform() {
  OS="$(uname -s)"
  case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)      fail "Unsupported operating system: $OS" ;;
  esac

  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             fail "Unsupported architecture: $ARCH" ;;
  esac

  PLATFORM="${OS}/${ARCH}"
}

# ─── Fetch Latest Version ───────────────────────────────────────────────────
fetch_version() {
  LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$LATEST" ]; then
    fail "Could not determine latest version. Check your internet connection."
  fi
}

# ─── Download Binary ─────────────────────────────────────────────────────────
download_binary() {
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}-${OS}-${ARCH}"
  TMPFILE=$(mktemp)

  HTTP_CODE=$(curl -fsSL -w "%{http_code}" "$DOWNLOAD_URL" -o "$TMPFILE" 2>/dev/null || echo "000")

  if [ "$HTTP_CODE" != "200" ] || [ ! -s "$TMPFILE" ]; then
    rm -f "$TMPFILE"
    fail "Download failed (HTTP ${HTTP_CODE}). Binary may not exist for ${PLATFORM}."
  fi

  success "Downloaded ${BINARY} ${LATEST} for ${PLATFORM}"
  chmod +x "$TMPFILE"
}

# ─── Install Binary ─────────────────────────────────────────────────────────
install_binary() {
  if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
  else
    warn "Elevated permissions required to install to ${INSTALL_DIR}"
    sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
  fi
  success "Installed to ${INSTALL_DIR}/${BINARY}"
}

# ─── Setup PATH ──────────────────────────────────────────────────────────────
setup_path() {
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) PATH_EXISTS=true ;;
    *)                     PATH_EXISTS=false ;;
  esac

  if $PATH_EXISTS; then
    success "${INSTALL_DIR} is already in your PATH"
    return
  fi

  EXPORT_LINE="export PATH=\"${INSTALL_DIR}:\$PATH\""
  CURRENT_SHELL="$(basename "${SHELL:-/bin/sh}")"

  case "$CURRENT_SHELL" in
    zsh)  SHELL_NAME="zsh" ;;
    bash) SHELL_NAME="bash" ;;
    fish) SHELL_NAME="fish" ;;
    *)    SHELL_NAME="$CURRENT_SHELL" ;;
  esac

  RC_FILES=""
  case "$SHELL_NAME" in
    zsh)
      [ -f "$HOME/.zshrc" ]    && RC_FILES="$HOME/.zshrc"
      [ -f "$HOME/.zprofile" ] && RC_FILES="${RC_FILES:+$RC_FILES }$HOME/.zprofile"
      [ -z "$RC_FILES" ]       && RC_FILES="$HOME/.zshrc"
      ;;
    bash)
      [ -f "$HOME/.bashrc" ]       && RC_FILES="$HOME/.bashrc"
      [ -f "$HOME/.bash_profile" ] && RC_FILES="${RC_FILES:+$RC_FILES }$HOME/.bash_profile"
      [ -f "$HOME/.profile" ]      && RC_FILES="${RC_FILES:+$RC_FILES }$HOME/.profile"
      [ -z "$RC_FILES" ]           && RC_FILES="$HOME/.bashrc"
      ;;
    fish)
      FISH_CONF="$HOME/.config/fish/config.fish"
      [ -d "$HOME/.config/fish" ] || mkdir -p "$HOME/.config/fish"
      RC_FILES="$FISH_CONF"
      ;;
    *)
      [ -f "$HOME/.profile" ] && RC_FILES="$HOME/.profile"
      ;;
  esac

  if [ -z "$RC_FILES" ]; then
    warn "Could not detect shell config file"
    warn "Manually add to your shell config: ${EXPORT_LINE}"
    return
  fi

  ADDED_TO=""
  for rc in $RC_FILES; do
    if [ -f "$rc" ] && grep -qF "$INSTALL_DIR" "$rc" 2>/dev/null; then
      continue
    fi
    if [ "$SHELL_NAME" = "fish" ]; then
      printf "\n# Added by hudl installer\nfish_add_path %s\n" "$INSTALL_DIR" >> "$rc"
    else
      printf "\n# Added by hudl installer\n%s\n" "$EXPORT_LINE" >> "$rc"
    fi
    ADDED_TO="${ADDED_TO:+$ADDED_TO, }$(basename "$rc")"
  done

  if [ -n "$ADDED_TO" ]; then
    success "Added ${INSTALL_DIR} to PATH in ${ADDED_TO}"
  fi

  PRIMARY_RC="$(echo "$RC_FILES" | awk '{print $1}')"
  printf "\n"
  info "To use ${BOLD}hudl${RESET} right now, run:"
  printf "\n"
  printf "    ${CYAN}\$ ${WHITE}source ${PRIMARY_RC}${RESET}\n"
  printf "\n"
  info "Or just open a ${BOLD}new terminal${RESET} window"
}

# ─── Verify Installation ────────────────────────────────────────────────────
verify() {
  if command -v "$BINARY" >/dev/null 2>&1; then
    INSTALLED_VERSION=$("$BINARY" --version 2>/dev/null || echo "unknown")
    success "Verified: ${INSTALLED_VERSION}"
  else
    setup_path
  fi
}

# ─── Post-Install ───────────────────────────────────────────────────────────
post_install() {
  printf "\n"
  printf "  ${DIM}─────────────────────────────────────────────${RESET}\n"
  printf "\n"
  printf "  ${BOLD}${WHITE}Get started:${RESET}\n"
  printf "\n"
  printf "    ${CYAN}\$ ${WHITE}hudl auth login${RESET}        ${DIM}# authenticate${RESET}\n"
  printf "    ${CYAN}\$ ${WHITE}hudl vm list${RESET}           ${DIM}# list instances${RESET}\n"
  printf "    ${CYAN}\$ ${WHITE}hudl --help${RESET}            ${DIM}# see all commands${RESET}\n"
  printf "\n"
  printf "  ${DIM}Docs:${RESET} ${CYAN}https://console.huddle01.com/docs/cli${RESET}\n"
  printf "\n"
}

# ─── Main ────────────────────────────────────────────────────────────────────
main() {
  banner

  printf "  ${DIM}─────────────────────────────────────────────${RESET}\n"
  printf "\n"

  detect_platform
  info "Detected platform: ${BOLD}${PLATFORM}${RESET}"

  fetch_version
  info "Latest version: ${BOLD}${LATEST}${RESET}"

  printf "\n"

  download_binary
  install_binary
  verify

  post_install
}

main "$@"
