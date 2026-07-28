package gamecore

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

const setupArtifactDigestDomain = "gamecore/setup-artifact/v1"

type SetupArtifact struct {
	descriptor Descriptor
	version    ArtifactVersion
	payload    []byte
	digest     Digest
}

func NewSetupArtifact(descriptor Descriptor, version ArtifactVersion, payload []byte) (SetupArtifact, error) {
	if err := descriptor.Validate(); err != nil {
		return SetupArtifact{}, err
	}
	if err := validateIdentifier("artifactVersion", string(version)); err != nil {
		return SetupArtifact{}, err
	}
	if err := validatePayload("artifact payload", payload, false); err != nil {
		return SetupArtifact{}, err
	}
	artifact := SetupArtifact{descriptor: descriptor, version: version, payload: cloneBytes(payload)}
	artifact.digest = artifact.computeDigest()
	return artifact, nil
}

func RestoreSetupArtifact(descriptor Descriptor, version ArtifactVersion, payload []byte, digest Digest) (SetupArtifact, error) {
	artifact, err := NewSetupArtifact(descriptor, version, payload)
	if err != nil {
		return SetupArtifact{}, err
	}
	if allZero(digest) || artifact.digest != digest {
		return SetupArtifact{}, fmt.Errorf("%w: setup artifact digest mismatch", ErrVerificationFailed)
	}
	return artifact, nil
}

func (a SetupArtifact) Validate() error {
	if err := a.descriptor.Validate(); err != nil {
		return err
	}
	if err := validateIdentifier("artifactVersion", string(a.version)); err != nil {
		return err
	}
	if err := validatePayload("artifact payload", a.payload, false); err != nil {
		return err
	}
	if allZero(a.digest) || a.computeDigest() != a.digest {
		return fmt.Errorf("%w: setup artifact digest mismatch", ErrVerificationFailed)
	}
	return nil
}

func (a SetupArtifact) Descriptor() Descriptor   { return a.descriptor }
func (a SetupArtifact) Version() ArtifactVersion { return a.version }
func (a SetupArtifact) Payload() []byte          { return cloneBytes(a.payload) }
func (a SetupArtifact) Digest() Digest           { return a.digest }

func (a SetupArtifact) computeDigest() Digest {
	h := sha256.New()
	writeDomain(h, setupArtifactDigestDomain)
	writeString(h, string(a.descriptor.GameID()))
	writeString(h, string(a.descriptor.RulesetVersion()))
	writeString(h, string(a.descriptor.ModuleVersion()))
	writeString(h, string(a.descriptor.FairnessSuiteID()))
	_, _ = h.Write([]byte{a.descriptor.ParticipantCount()})
	writeString(h, string(a.version))
	writeBytes(h, a.payload)
	var digest Digest
	copy(digest[:], h.Sum(nil))
	return digest
}

type RandomizedSetupModule interface {
	Descriptor() Descriptor
	GenerateSetup(FairnessMaterial) (SetupArtifact, error)
	VerifySetup(FairnessMaterial, SetupArtifact) error
}

type registeredModule struct {
	descriptor Descriptor
	delegate   RandomizedSetupModule
}

func (m registeredModule) Descriptor() Descriptor { return m.descriptor }

func (m registeredModule) GenerateSetup(material FairnessMaterial) (SetupArtifact, error) {
	if !material.Descriptor.Equal(m.descriptor) {
		return SetupArtifact{}, fmt.Errorf("%w: setup material descriptor mismatch", ErrInvalidArgument)
	}
	artifact, err := m.delegate.GenerateSetup(material.Clone())
	if err != nil {
		return SetupArtifact{}, err
	}
	if err := artifact.Validate(); err != nil {
		return SetupArtifact{}, err
	}
	if !artifact.Descriptor().Equal(m.descriptor) {
		return SetupArtifact{}, fmt.Errorf("%w: module returned mismatched artifact descriptor", ErrVerificationFailed)
	}
	return artifact, nil
}

func (m registeredModule) VerifySetup(material FairnessMaterial, artifact SetupArtifact) error {
	if !material.Descriptor.Equal(m.descriptor) || !artifact.Descriptor().Equal(m.descriptor) {
		return fmt.Errorf("%w: setup identity mismatch", ErrInvalidArgument)
	}
	if err := artifact.Validate(); err != nil {
		return err
	}
	return m.delegate.VerifySetup(material.Clone(), artifact)
}

type Registry struct {
	mu      sync.RWMutex
	modules map[DescriptorKey]registeredModule
}

func NewRegistry(modules ...RandomizedSetupModule) (*Registry, error) {
	registry := &Registry{modules: make(map[DescriptorKey]registeredModule, len(modules))}
	for _, module := range modules {
		if err := registry.Register(module); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(module RandomizedSetupModule) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrInvalidArgument)
	}
	if module == nil {
		return fmt.Errorf("%w: nil setup module", ErrInvalidArgument)
	}
	descriptor := module.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return err
	}
	key := descriptor.Key()
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.modules[key]; exists {
		return fmt.Errorf("%w: %s/%s/%s", ErrDuplicateRegistration, key.GameID, key.RulesetVersion, key.ModuleVersion)
	}
	r.modules[key] = registeredModule{descriptor: descriptor, delegate: module}
	return nil
}

func (r *Registry) Lookup(key DescriptorKey) (RandomizedSetupModule, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: nil registry", ErrInvalidArgument)
	}
	if err := validateIdentifier("gameId", string(key.GameID)); err != nil {
		return nil, err
	}
	if err := validateIdentifier("rulesetVersion", string(key.RulesetVersion)); err != nil {
		return nil, err
	}
	if err := validateIdentifier("moduleVersion", string(key.ModuleVersion)); err != nil {
		return nil, err
	}
	r.mu.RLock()
	module, exists := r.modules[key]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s/%s/%s", ErrModuleNotFound, key.GameID, key.RulesetVersion, key.ModuleVersion)
	}
	return module, nil
}

func (r *Registry) Count() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.modules)
}
