# Goal0063 Round5 evidence

Fixed source: `5d0f6157cfe724bf960dd66d1f6572787f96a330`. Final **FAIL**, R4-N01 PASS, R4-N02 FAIL. See the report and summary.json for the evidence mapping.

- native-check.json is retained raw: desktop hand cases and room UNKNOWN observations pass; its early hand-adoption and non-projected geometry FAILs are fixture issues explained in fixture-notes.md. Its process was intentionally stopped (-15) to correct the fixture.
- resume.json uses actual camera-projected screen rectangles: mobile Play/Pass and narrow Play pass; narrow Pass fails on PublicTrick overlap. exit0 means the runner finished, not product PASS.
- followups.json contains seven passing actual/controlled-native boundary checks. Metadata-only hand/purpose checks are identified, with a separate real next-hand adoption check.
- visual-observer.json is a passive live Cocos geometry observation, not a synthetic layout. overlap-760x390.json and the two narrow Pass screenshots establish the remaining product defect. All published screenshot dimensions were checked.
- binding.json distinguishes original build artifact hashes from the runtime configuration/HTML reference injection. CI used a merge-test SHA with identical build inputs, not a candidate-commit checkout claim. Build warnings are retained; the final real Cocos4 declaration/JS audit passed.
- cleanup.json records stopped owned services and retained data; optional cancellation's already-terminal rejection is retained in fixture-finish.json. No Kubernetes mutation, merge, product fix or original Goal state change.

Reproduction commands, with E=/root/goal0063-round5-5d0f615, use pinned Node22.23.4 and private fixture configuration: native-build.cjs (official CLI), focused-final command (strict shell compile + two affected stock files, 25 tests), native-check.cjs, resume.cjs, boundaries.cjs and inspect-overlap.cjs. The public text copies record exact assertions. External API, TLS, QA accounts and dependencies are private and not included. Source fixes belong to ChatGPT.

No passwords, JWTs, private keys, full browser profiles, private runtime configuration or binaries are published. Only dummy personal draft strings and isolated QA game data/screenshots are included. The known-secret publication scan was bounded; it is not a general proof about all machine logs. sha256.json covers the published files except itself.

Published logs trim trailing whitespace only; private originals are retained.
