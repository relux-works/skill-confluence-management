#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SKILL_NAME="confluence-management"
SKILL_CONTENT_DIR="$PROJECT_ROOT/agents/skills/$SKILL_NAME"
BINARY_NAME="confluence-mgmt"
BIN_DIR="${CONFLUENCE_MGMT_BIN_DIR:-$HOME/.local/bin}"
INSTALL_ONLY="${CONFLUENCE_MGMT_INSTALL_ONLY:-0}"

AGENTS_DIR="$HOME/.agents/skills"
CLAUDE_DIR="$HOME/.claude/skills"
CODEX_DIR="$HOME/.codex/skills"
INSTALLED_SKILL_DIR="$AGENTS_DIR/$SKILL_NAME"
INSTALLED_BIN="$BIN_DIR/$BINARY_NAME"

BUILD_VERSION="dev"
BUILD_COMMIT="unknown"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_LDFLAGS=""

ARTIFACT_DIR="$(mktemp -d "${TMPDIR:-/tmp}/confluence-management-setup.XXXXXX")"
ARTIFACT_SKILL_DIR="$ARTIFACT_DIR/skill"
ARTIFACT_BIN="$ARTIFACT_DIR/$BINARY_NAME"

cleanup() {
  rm -rf "$ARTIFACT_DIR"
}

trap cleanup EXIT

green() { printf '\033[32m%s\033[0m\n' "$1"; }
yellow() { printf '\033[33m%s\033[0m\n' "$1"; }
red() { printf '\033[31m%s\033[0m\n' "$1"; }

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

config_dir() {
  case "$(uname -s)" in
    Darwin) printf '%s' "$HOME/Library/Application Support/confluence-mgmt" ;;
    *) printf '%s' "${XDG_CONFIG_HOME:-$HOME/.config}/confluence-mgmt" ;;
  esac
}

usage() {
  cat <<EOF
Usage: scripts/setup.sh [options]

Options:
  --bin-dir PATH       Install binary into PATH (default: $HOME/.local/bin)
  --install-only       Safe reinstall of binary, skill artifact, links, and install metadata
  --help, -h           Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bin-dir)
      BIN_DIR="$2"
      shift 2
      ;;
    --install-only)
      INSTALL_ONLY="1"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      red "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
done

install_go() {
  if command -v go >/dev/null 2>&1; then
    green "Go already installed: $(go version)"
    return
  fi

  if [[ "$(uname -s)" != "Darwin" ]]; then
    red "Go is missing. Install Go first, then rerun setup."
    exit 1
  fi

  if ! command -v brew >/dev/null 2>&1; then
    red "Go is missing and Homebrew is not available. Install Homebrew or Go first."
    exit 1
  fi

  yellow "Go not found. Installing via Homebrew..."
  brew install go
  green "Go installed: $(go version)"
}

compute_ldflags() {
  if git -C "$PROJECT_ROOT" rev-parse --git-dir >/dev/null 2>&1; then
    BUILD_VERSION="$(git -C "$PROJECT_ROOT" describe --tags --always 2>/dev/null || echo "dev")"
    BUILD_COMMIT="$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")"
  fi

  BUILD_LDFLAGS="-X main.version=$BUILD_VERSION -X main.commit=$BUILD_COMMIT -X main.date=$BUILD_DATE"
}

build_cli() {
  green "Building $BINARY_NAME ..."
  (
    cd "$PROJECT_ROOT"
    go build -trimpath -ldflags "$BUILD_LDFLAGS" -o "$ARTIFACT_BIN" ./cmd/confluence-mgmt
  )
  green "Built artifact: $ARTIFACT_BIN"
}

install_binary() {
  mkdir -p "$BIN_DIR"
  cp "$ARTIFACT_BIN" "$INSTALLED_BIN"
  chmod +x "$INSTALLED_BIN"
  green "Installed binary: $INSTALLED_BIN"
}

install_skill_artifact() {
  mkdir -p "$ARTIFACT_SKILL_DIR"
  rsync -a --delete "$SKILL_CONTENT_DIR/" "$ARTIFACT_SKILL_DIR/" --exclude='.git'

  mkdir -p "$INSTALLED_SKILL_DIR"
  rsync -a --delete "$ARTIFACT_SKILL_DIR/" "$INSTALLED_SKILL_DIR/"
  green "Installed skill artifact: $INSTALLED_SKILL_DIR"
}

refresh_links() {
  mkdir -p "$CLAUDE_DIR" "$CODEX_DIR"
  rm -rf "$CLAUDE_DIR/$SKILL_NAME" "$CODEX_DIR/$SKILL_NAME"
  ln -s "$INSTALLED_SKILL_DIR" "$CLAUDE_DIR/$SKILL_NAME"
  ln -s "$INSTALLED_SKILL_DIR" "$CODEX_DIR/$SKILL_NAME"
  green "Refreshed Claude/Codex skill links"
}

write_install_state() {
  local config_dir_path install_state_path escaped_repo escaped_skill escaped_bin escaped_platform escaped_arch escaped_version escaped_commit escaped_build_date
  config_dir_path="$(config_dir)"
  install_state_path="$config_dir_path/install.json"
  mkdir -p "$config_dir_path"

  escaped_repo="$(json_escape "$PROJECT_ROOT")"
  escaped_skill="$(json_escape "$INSTALLED_SKILL_DIR")"
  escaped_bin="$(json_escape "$BIN_DIR")"
  escaped_platform="$(json_escape "$(uname -s | tr '[:upper:]' '[:lower:]')")"
  escaped_arch="$(json_escape "$(uname -m)")"
  escaped_version="$(json_escape "$BUILD_VERSION")"
  escaped_commit="$(json_escape "$BUILD_COMMIT")"
  escaped_build_date="$(json_escape "$BUILD_DATE")"

  cat > "$install_state_path" <<EOF
{
  "repoPath": "$escaped_repo",
  "installedSkillPath": "$escaped_skill",
  "binDir": "$escaped_bin",
  "platform": "$escaped_platform",
  "arch": "$escaped_arch",
  "version": "$escaped_version",
  "commit": "$escaped_commit",
  "buildDate": "$escaped_build_date",
  "installOnly": $([[ "$INSTALL_ONLY" == "1" ]] && echo "true" || echo "false")
}
EOF
  green "Install state: $install_state_path"
}

verify_install() {
  [[ -x "$INSTALLED_BIN" ]] || { red "Missing installed binary: $INSTALLED_BIN"; exit 1; }
  [[ -f "$INSTALLED_SKILL_DIR/SKILL.md" ]] || { red "Installed skill artifact is missing SKILL.md"; exit 1; }

  local resolved=""
  if resolved="$(command -v "$BINARY_NAME" 2>/dev/null)"; then
    if [[ "$resolved" != "$INSTALLED_BIN" ]]; then
      yellow "$BINARY_NAME on PATH resolves to $resolved"
      yellow "Expected: $INSTALLED_BIN"
    fi
  else
    yellow "$BIN_DIR is not in PATH yet."
    yellow "Add to your shell profile: export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi

  "$INSTALLED_BIN" version >/dev/null
  "$INSTALLED_BIN" auth config-path >/dev/null
  green "Verified binary and skill artifact"
}

printf "\n"
green "=== confluence-management setup ==="
printf "\n"
if [[ "$INSTALL_ONLY" == "1" ]]; then
  yellow "Running safe reinstall flow (--install-only)"
fi

install_go
compute_ldflags
build_cli
install_binary
install_skill_artifact
refresh_links
write_install_state
verify_install

printf "\n"
green "Next steps:"
printf "  confluence-mgmt auth set-access --instance URL --email EMAIL --token TOKEN\n"
printf "  confluence-mgmt auth whoami\n"
