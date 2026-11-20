# Goal0063 Round4 evidence

Source: `871ac82be1bce7d3e08e5068d7f46e84f920a76d`. Overall FAIL; R3-N01 passes, R4-N01/R4-N02 remain.

`native-check.json` contains four passing real hand cases, room state observations and a retained early create fixture FAIL (sent0: already managing an automatically recovered room). Its final PASS refers to the subsequent hand run, not all historical entries. Actual valid create input loss is the first FAIL in `followups.json`; later two actual conflict/isolation cases pass. A zero process exit code means the runner completed, not product acceptance. The first room start was stale and rejected; a refreshed accepted start is separately recorded in `room-sql.json` and native cases.

Source for the exact native assertions/hooks and four controlled state probes is included as text. Scripts use private fixtures in `/root/goal0063-round4-871ac82`; passwords, tokens, private keys/configs and browser profiles are excluded. Receipts/state summaries are QA-only. Screenshots are actual official Cocos Web pages and dimensions were checked. Native-build.log has trailing CLI whitespace removed for publication; raw log remains in the private evidence directory.

No full CI, server fault matrix, full gameplay or phone/device qualification was rerun. Native source binding, CI metadata and reused backend binary hash are recorded; cleanup stops owned runtime and test containers while retaining data.
