#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== confluence-mgmt Setup ==="

# 1. Build binary
echo "Building confluence-mgmt binary..."
cd "$PROJECT_ROOT"
go build -o confluence-mgmt ./cmd/confluence-mgmt/

# 2. Create ~/.local/bin if it doesn't exist
mkdir -p "$HOME/.local/bin"

# 3. Symlink binary to ~/.local/bin
echo "Creating symlink to ~/.local/bin/confluence-mgmt..."
ln -sf "$PROJECT_ROOT/confluence-mgmt" "$HOME/.local/bin/confluence-mgmt"

# 4. Verify PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
  echo ""
  echo "WARNING: ~/.local/bin is not in your PATH"
  echo "Add this to your ~/.zshrc:"
  echo '  export PATH="$HOME/.local/bin:$PATH"'
  echo ""
fi

# 5. Create skill symlinks
echo "Creating skill symlinks..."

# Claude Code symlink
mkdir -p "$HOME/.claude/skills"
ln -sf "$PROJECT_ROOT/agents/skills/confluence-management" "$HOME/.claude/skills/confluence-management"

# Codex CLI symlink
mkdir -p "$HOME/.codex/skills"
ln -sf "$PROJECT_ROOT/agents/skills/confluence-management" "$HOME/.codex/skills/confluence-management"

echo ""
echo "Setup complete!"
echo "Binary: $HOME/.local/bin/confluence-mgmt"
echo "Skill (Claude): ~/.claude/skills/confluence-management"
echo "Skill (Codex): ~/.codex/skills/confluence-management"
echo ""
echo "Next steps:"
echo "  1. Ensure ~/.local/bin is in your PATH"
echo "  2. Run: confluence-mgmt auth"
echo "  3. Configure: confluence-mgmt config set space YOUR-KEY"
