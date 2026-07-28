from pathlib import Path


def replace_once(text: str, old: str, new: str, name: str) -> str:
    if old not in text:
        if new in text:
            return text
        raise SystemExit(f"{name} marker not found")
    return text.replace(old, new, 1)


bootstrap = Path("server/apps/app-api/internal/bootstrap/bootstrap_integration_test.go")
text = bootstrap.read_text()
text = text.replace('schemaVersion != "0010"', 'schemaVersion != "0011"')
text = text.replace('schema version = %q, want 0010', 'schema version = %q, want 0011')
marker = '\t\t"platform_audit_events",\n'
if '\t\t"game_final_records",\n' not in text:
    text = replace_once(text, marker, marker + '\t\t"game_final_records",\n', "bootstrap table")
bootstrap.write_text(text)

integration = Path("server/business/doudizhu/infrastructure/mysqlstore/integration_test.go")
text = integration.read_text()
import_marker = '\t"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"\n'
if 'server/business/gamecore"' not in text:
    text = replace_once(
        text,
        import_marker,
        import_marker + '\t"github.com/oublie6/awesome-zero-platform/server/business/gamecore"\n',
        "gamecore import",
    )

text = replace_once(
    text,
    'application.NewService(store, clock, ids, integrationSetup{}, opener, protector, textnormalization.NFKC{}, application.DefaultConfig())',
    'application.NewService(store, clock, ids, integrationSetup{}, integrationBeaconVerifier{}, integrationLiveRuntime{}, opener, protector, textnormalization.NFKC{}, application.DefaultConfig())',
    "inline NewService",
)
text = replace_once(
    text,
    '''\t\tstore, clock, &integrationIDs{prefix: prefix}, integrationSetup{},
\t\t&integrationOpener{}, protector, textnormalization.NFKC{}, application.DefaultConfig(),
''',
    '''\t\tstore, clock, &integrationIDs{prefix: prefix}, integrationSetup{},
\t\tintegrationBeaconVerifier{}, integrationLiveRuntime{},
\t\t&integrationOpener{}, protector, textnormalization.NFKC{}, application.DefaultConfig(),
''',
    "helper NewService",
)

setup_marker = '''func (integrationSetup) PrepareHand(context.Context, domain.RoomSnapshot, domain.HandID) (application.HandSetup, error) {
\treturn application.HandSetup{}, fmt.Errorf("not used")
}
'''
if "func (integrationSetup) ReleaseHand" not in text:
    text = replace_once(
        text,
        setup_marker,
        setup_marker + '\nfunc (integrationSetup) ReleaseHand(context.Context, domain.HandID) error { return nil }\n',
        "ReleaseHand",
    )

opener_marker = 'type integrationOpener struct {\n'
stubs = '''type integrationBeaconVerifier struct{}

func (integrationBeaconVerifier) Verify(_ context.Context, _ domain.BeaconPlan, value domain.BeaconValue) (domain.BeaconValue, error) {
\treturn value, nil
}

type integrationLiveRuntime struct{}

func (integrationLiveRuntime) Start(context.Context, domain.HandSnapshot) error { return fmt.Errorf("not used") }
func (integrationLiveRuntime) RollbackStart(context.Context, domain.HandID) error { return nil }
func (integrationLiveRuntime) ReleasePrepared(context.Context, domain.HandID) error { return nil }
func (integrationLiveRuntime) PublicView(context.Context, domain.HandID, domain.AccountID) (application.LiveHandView, error) {
\treturn application.LiveHandView{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) PrivateView(context.Context, domain.HandID, domain.AccountID) (application.LiveHandView, error) {
\treturn application.LiveHandView{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) Abort(context.Context, domain.HandID, string) (gamecore.FinalRecord, error) {
\treturn gamecore.FinalRecord{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) RetryArchive(context.Context, domain.HandID) (gamecore.FinalRecord, error) {
\treturn gamecore.FinalRecord{}, fmt.Errorf("not used")
}
func (integrationLiveRuntime) Contains(domain.HandID) bool { return false }

'''
if "type integrationBeaconVerifier struct{}" not in text:
    text = replace_once(text, opener_marker, stubs + opener_marker, "integration runtime stubs")

required = [
    "integrationBeaconVerifier{}",
    "integrationLiveRuntime{}",
    "func (integrationSetup) ReleaseHand",
    "type integrationBeaconVerifier struct{}",
]
missing = [value for value in required if value not in text]
if missing:
    raise SystemExit(f"integration repair incomplete: {missing}")
integration.write_text(text)
