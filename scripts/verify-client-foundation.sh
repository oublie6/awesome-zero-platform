#!/usr/bin/env bash
# Portable source checks; this does not run Creator or a mini-game developer tool.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
for package in game-client-core doudizhu-client; do
  npm --prefix "clients/packages/$package" ci --no-audit --no-fund
  npm --prefix "clients/packages/$package" run typecheck
  npm --prefix "clients/packages/$package" test
done
npm --prefix clients/packages/secure-envelope install --package-lock=false --no-audit --no-fund
npm --prefix clients/packages/secure-envelope run build
npm --prefix clients/fair-doudizhu-cocos ci --no-audit --no-fund
npm --prefix clients/fair-doudizhu-cocos run typecheck
npm --prefix clients/fair-doudizhu-cocos run typecheck:shell
npm --prefix clients/fair-doudizhu-cocos test
