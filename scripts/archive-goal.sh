#!/usr/bin/env bash
# Archive a completed goal and reset current.md from the template.
set -euo pipefail

GOAL_NAME="${1:-}"
if [[ -z "${GOAL_NAME}" ]]; then
  echo "Usage: $0 <goal-name>" >&2
  exit 1
fi

if [[ ! "${GOAL_NAME}" =~ ^[a-z0-9][a-z0-9._-]*$ ]]; then
  echo "Error: goal name must use lowercase letters, numbers, dots, underscores, or hyphens." >&2
  exit 1
fi

if ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null); then
  :
else
  echo "Error: run this script inside the repository." >&2
  exit 1
fi

cd "${ROOT_DIR}"

CURRENT="docs/goals/current.md"
TEMPLATE="docs/goals/template.md"
ARCHIVE_DIR="docs/goals/archive"
ARCHIVE="${ARCHIVE_DIR}/${GOAL_NAME}.md"

[[ -f "${CURRENT}" ]] || { echo "Error: ${CURRENT} not found." >&2; exit 1; }
[[ -f "${TEMPLATE}" ]] || { echo "Error: ${TEMPLATE} not found." >&2; exit 1; }

if ! grep -Eq '^- State: completed[[:space:]]*$' "${CURRENT}"; then
  echo "Error: current goal is not marked completed." >&2
  exit 1
fi

if [[ -e "${ARCHIVE}" ]]; then
  echo "Error: archive already exists: ${ARCHIVE}" >&2
  exit 1
fi

mkdir -p "${ARCHIVE_DIR}"
mv "${CURRENT}" "${ARCHIVE}"
cp "${TEMPLATE}" "${CURRENT}"

echo "Archived completed goal to ${ARCHIVE}"
echo "Reset ${CURRENT} from ${TEMPLATE}"
echo "Archived goals are retained until deliberate cleanup is needed."
