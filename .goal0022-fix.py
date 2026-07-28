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
