#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SKILL_NAME="confluence-management"
SKILL_CONTENT_DIR="$PROJECT_ROOT/agents/skills/$SKILL_NAME"

AGENTS_DIR="$HOME/.agents/skills"
CLAUDE_DIR="$HOME/.claude/skills"
CODEX_DIR="$HOME/.codex/skills"

echo "=== $SKILL_NAME Setup ==="

# 1. Build binary
echo "Building confluence-mgmt binary..."
cd "$PROJECT_ROOT"
go build -o confluence-mgmt ./cmd/confluence-mgmt/

# 2. Symlink binary to ~/.local/bin
mkdir -p "$HOME/.local/bin"
ln -sf "$PROJECT_ROOT/confluence-mgmt" "$HOME/.local/bin/confluence-mgmt"
echo "  Binary -> ~/.local/bin/confluence-mgmt"

# 3. Verify PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
  echo "  WARNING: ~/.local/bin is not in your PATH"
fi

# 4. Copy skill into .agents/skills/ (installed copy, not a symlink)
echo "Installing skill: $SKILL_NAME"
if [ -L "$AGENTS_DIR/$SKILL_NAME" ]; then
  rm -f "$AGENTS_DIR/$SKILL_NAME"
fi
mkdir -p "$AGENTS_DIR/$SKILL_NAME"
rsync -a --delete "$SKILL_CONTENT_DIR/" "$AGENTS_DIR/$SKILL_NAME/" --exclude='.git'
echo "  Copied -> $AGENTS_DIR/$SKILL_NAME/"

# 5. Symlink from .claude/skills/ -> .agents/skills/
mkdir -p "$CLAUDE_DIR"
rm -f "$CLAUDE_DIR/$SKILL_NAME"
ln -s "$AGENTS_DIR/$SKILL_NAME" "$CLAUDE_DIR/$SKILL_NAME"
echo "  Symlink -> $CLAUDE_DIR/$SKILL_NAME"

# 6. Symlink from .codex/skills/ -> .agents/skills/
mkdir -p "$CODEX_DIR"
rm -f "$CODEX_DIR/$SKILL_NAME"
ln -s "$AGENTS_DIR/$SKILL_NAME" "$CODEX_DIR/$SKILL_NAME"
echo "  Symlink -> $CODEX_DIR/$SKILL_NAME"

echo ""
echo "Done. Installed $(git -C "$PROJECT_ROOT" describe --tags --always 2>/dev/null || echo 'unknown')"
echo ""
echo "Next steps:"
echo "  confluence-mgmt auth"
echo "  confluence-mgmt config set space YOUR-KEY"
