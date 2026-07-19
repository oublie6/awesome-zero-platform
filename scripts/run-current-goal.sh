#!/usr/bin/env bash
# Start a fresh Codex session for the active repository goal.
set -euo pipefail

if ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null); then
  :
else
  echo "Error: run this script inside the repository." >&2
  exit 1
fi

cd "${ROOT_DIR}"

if [[ ! -f AGENTS.md ]]; then
  echo "Error: AGENTS.md not found." >&2
  exit 1
fi

if [[ ! -f docs/goals/current.md ]]; then
  echo "Error: docs/goals/current.md not found." >&2
  exit 1
fi

if ! command -v codex >/dev/null 2>&1; then
  echo "Error: codex command not found. Install Codex CLI and ensure it is in PATH." >&2
  exit 1
fi

if grep -Eq '^- State: idle[[:space:]]*$' docs/goals/current.md; then
  echo "Error: current goal is idle. Define a concrete goal before starting Codex." >&2
  exit 1
fi

PROMPT=$(cat <<'EOF'
Read AGENTS.md and docs/goals/current.md completely before doing anything.
Treat them as authoritative repository instructions.
Read only the architecture, requirement, and decision documents explicitly referenced by the current goal, plus source files required to implement and verify it.
Do not load archived goals unless the current goal explicitly requires historical investigation.
Do not expand or reinterpret the goal, deliverables, constraints, or acceptance criteria.
Use subagents only for independent work and do not allow multiple agents to modify the same files concurrently.
Complete the current goal, run every acceptance check, and update only the permitted status, working-state, and completion-report sections before stopping.
EOF
)

exec env TERM=screen-256color COLORTERM=truecolor \
  codex --dangerously-bypass-approvals-and-sandbox "${PROMPT}"
