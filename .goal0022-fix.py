from pathlib import Path

path = Path('server/business/doudizhu/infrastructure/securetransport/opener_test.go')
text = path.read_text()
old = '''\tif status == revealkeys.StatusActive {
\t\tactive.Status = revealkeys.StatusRetired
\t\tactive.RetiringAt = now.Add(-2 * time.Hour)
\t\tactive.RetireAfter = now.Add(-time.Hour)
\t\tcurrentKeyID = "bound-key"
\t}
'''
new = '''\tif status == revealkeys.StatusActive {
\t\tbound.ManifestVersion = 2
\t\tactive.ManifestVersion = 1
\t\tactive.Status = revealkeys.StatusRetired
\t\tactive.RetiringAt = now.Add(-2 * time.Hour)
\t\tactive.RetireAfter = now.Add(-time.Hour)
\t\tcurrentKeyID = "bound-key"
\t}
'''
if old not in text:
    raise SystemExit('Goal 0022 fix patch did not match opener_test.go')
path.write_text(text.replace(old, new, 1))

manifest_path = Path('clients/packages/secure-envelope/src/manifest.ts')
manifest = manifest_path.read_text()
old_import = "import { SUITE_V1 } from './constants.js'"
old_usage = 'manifest.suite !== SUITE_V1'
if old_import not in manifest:
    raise SystemExit('Goal 0022 fix patch did not find SUITE_V1 import')
if old_usage not in manifest:
    raise SystemExit('Goal 0022 fix patch did not find SUITE_V1 usage')
manifest = manifest.replace(
    old_import,
    "import { SECURE_ENVELOPE_SUITE } from './constants.js'",
    1,
)
manifest = manifest.replace(
    old_usage,
    'manifest.suite !== SECURE_ENVELOPE_SUITE',
    1,
)
manifest_path.write_text(manifest)
