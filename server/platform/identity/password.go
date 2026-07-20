package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(encodedHash string, password string) (bool, error)
	Params() PasswordParams
}

var DefaultPasswordParams = PasswordParams{
	MemoryKiB:   64 * 1024,
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

var TestPasswordParams = PasswordParams{
	MemoryKiB:   8 * 1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

type Argon2idHasher struct {
	params PasswordParams
	random io.Reader
}

func NewArgon2idHasher() *Argon2idHasher {
	return &Argon2idHasher{
		params: DefaultPasswordParams,
		random: rand.Reader,
	}
}

func NewTestArgon2idHasher() *Argon2idHasher {
	return &Argon2idHasher{
		params: TestPasswordParams,
		random: rand.Reader,
	}
}

func (h *Argon2idHasher) Hash(password string) (string, error) {
	if h == nil {
		return "", fmt.Errorf("password hasher is not configured")
	}

	salt := make([]byte, h.params.SaltLength)
	if _, err := io.ReadFull(h.random, salt); err != nil {
		return "", fmt.Errorf("read password salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, h.params.Iterations, h.params.MemoryKiB, h.params.Parallelism, h.params.KeyLength)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.MemoryKiB,
		h.params.Iterations,
		h.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func (h *Argon2idHasher) Verify(encodedHash string, password string) (bool, error) {
	params, salt, hash, err := parseArgon2idHash(encodedHash)
	if err != nil {
		return false, err
	}

	comparison := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKiB, params.Parallelism, params.KeyLength)

	return subtle.ConstantTimeCompare(hash, comparison) == 1, nil
}

func (h *Argon2idHasher) Params() PasswordParams {
	if h == nil {
		return PasswordParams{}
	}

	return h.params
}

func parseArgon2idHash(encodedHash string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return PasswordParams{}, nil, nil, fmt.Errorf("password hash format is invalid")
	}

	versionPart := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.Atoi(versionPart)
	if err != nil || version != argon2.Version {
		return PasswordParams{}, nil, nil, fmt.Errorf("password hash version is invalid")
	}

	var params PasswordParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.MemoryKiB, &params.Iterations, &params.Parallelism); err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("password hash parameters are invalid")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("password hash salt is invalid")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("password hash value is invalid")
	}

	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}
