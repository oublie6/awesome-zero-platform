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

GOAL_STATE=$(sed -n 's/^- State:[[:space:]]*//p' docs/goals/current.md | head -n 1)
case "${GOAL_STATE}" in
  ready|in_progress)
    ;;
  idle)
    echo "Error: current goal is idle. Define a concrete goal before starting Codex." >&2
    exit 1
    ;;
  completed)
    echo "Error: current goal is completed. Archive it and define the next goal before starting Codex." >&2
    exit 1
    ;;
  blocked)
    echo "Error: current goal is blocked. Resolve or replace the documented blocker before restarting Codex." >&2
    exit 1
    ;;
  "")
    echo "Error: current goal has no State value." >&2
    exit 1
    ;;
  *)
    echo "Error: unsupported goal state '${GOAL_STATE}'. Expected one of: idle, ready, in_progress, completed, blocked." >&2
    exit 1
    ;;
esac

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Error: working tree is not clean. Commit, push, or preserve existing work before starting a new goal." >&2
  git status --short >&2
  exit 1
fi

CURRENT_BRANCH=$(git branch --show-current)
if [[ -z "${CURRENT_BRANCH}" ]]; then
  echo "Error: detached HEAD is not supported for goal execution." >&2
  exit 1
fi

if ! git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' >/dev/null 2>&1; then
  echo "Error: branch '${CURRENT_BRANCH}' has no configured upstream." >&2
  exit 1
fi

echo "==> Synchronizing ${CURRENT_BRANCH} with its upstream"
git pull --ff-only

PROMPT=$(cat <<'EOF'
Read AGENTS.md and docs/goals/current.md completely before doing anything.
Treat them as authoritative repository instructions.
Confirm the repository is synchronized and clean before editing.
Read only the architecture, requirement, and decision documents explicitly referenced by the current goal, plus source files required to implement and verify it.
Do not load archived goals unless the current goal explicitly requires historical investigation.
Do not expand or reinterpret the goal, deliverables, constraints, or acceptance criteria.
Use subagents only for independent work and do not allow multiple agents to modify the same files concurrently.
Complete the current goal, run every acceptance check, and update only the permitted status, working-state, and completion-report sections.
Inspect the final diff, commit all goal-related changes, and push the current branch to its configured upstream without force pushing.
If synchronization, commit, or push cannot be completed safely, stop and document the exact blocker while preserving the local work.
EOF
)

exec env TERM=screen-256color COLORTERM=truecolor \
  codex --dangerously-bypass-approvals-and-sandbox "${PROMPT}"
