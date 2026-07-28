package gamecore

import "fmt"

type GameID string
type RulesetVersion string
type ModuleVersion string
type FairnessSuiteID string
type ArtifactVersion string
type InstanceID string

const MaxParticipantCount uint8 = 64

type Descriptor struct {
	gameID           GameID
	rulesetVersion   RulesetVersion
	moduleVersion    ModuleVersion
	fairnessSuiteID  FairnessSuiteID
	participantCount uint8
}

func NewDescriptor(gameID GameID, ruleset RulesetVersion, module ModuleVersion, fairness FairnessSuiteID, participantCount uint8) (Descriptor, error) {
	descriptor := Descriptor{
		gameID:           gameID,
		rulesetVersion:   ruleset,
		moduleVersion:    module,
		fairnessSuiteID:  fairness,
		participantCount: participantCount,
	}
	if err := descriptor.Validate(); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func (d Descriptor) Validate() error {
	if err := validateIdentifier("gameId", string(d.gameID)); err != nil {
		return err
	}
	if err := validateIdentifier("rulesetVersion", string(d.rulesetVersion)); err != nil {
		return err
	}
	if err := validateIdentifier("moduleVersion", string(d.moduleVersion)); err != nil {
		return err
	}
	if err := validateIdentifier("fairnessSuiteId", string(d.fairnessSuiteID)); err != nil {
		return err
	}
	if d.participantCount < 1 || d.participantCount > MaxParticipantCount {
		return fmt.Errorf("%w: participant count %d", ErrInvalidArgument, d.participantCount)
	}
	return nil
}

func (d Descriptor) GameID() GameID                   { return d.gameID }
func (d Descriptor) RulesetVersion() RulesetVersion   { return d.rulesetVersion }
func (d Descriptor) ModuleVersion() ModuleVersion     { return d.moduleVersion }
func (d Descriptor) FairnessSuiteID() FairnessSuiteID { return d.fairnessSuiteID }
func (d Descriptor) ParticipantCount() uint8          { return d.participantCount }

func (d Descriptor) Equal(other Descriptor) bool { return d == other }

func validateInstanceID(id InstanceID) error {
	return validateIdentifier("instanceId", string(id))
}

type DescriptorKey struct {
	GameID         GameID
	RulesetVersion RulesetVersion
	ModuleVersion  ModuleVersion
}

func (d Descriptor) Key() DescriptorKey {
	return DescriptorKey{GameID: d.gameID, RulesetVersion: d.rulesetVersion, ModuleVersion: d.moduleVersion}
}
