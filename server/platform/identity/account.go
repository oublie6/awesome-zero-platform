package identity

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	AccountIDLength      = 36
	DisplayNameMaxLength = 120
	UsernameMinLength    = 3
	UsernameMaxLength    = 32
	EmailMaxLength       = 320
	PhoneMaxLength       = 16
	PasswordMinBytes     = 8
	PasswordMaxBytes     = 128
)

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,31}$`)
	phonePattern    = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
)

type AccountStatus string

const (
	StatusActive   AccountStatus = "active"
	StatusDisabled AccountStatus = "disabled"
)

type Identity struct {
	Username string
	Email    string
	Phone    string
}

type NormalizedIdentity struct {
	Username    string
	UsernameKey string
	Email       string
	EmailKey    string
	Phone       string
	PhoneKey    string
}

type Account struct {
	ID          string
	Username    string
	Email       string
	Phone       string
	DisplayName string
	Status      AccountStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PasswordCredential struct {
	AccountID         string
	PasswordChangedAt time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateAccountInput struct {
	Identity    Identity
	DisplayName string
	Status      AccountStatus
	Password    string
}

type UpdateProfileInput struct {
	Username    *string
	Email       *string
	Phone       *string
	DisplayName *string
}

func (s AccountStatus) Validate() error {
	switch s {
	case StatusActive, StatusDisabled:
		return nil
	default:
		return fmt.Errorf("account status must be active or disabled")
	}
}

func (a Account) Validate() error {
	if _, err := normalizeAccountID(a.ID); err != nil {
		return err
	}

	normalized, err := Identity{
		Username: a.Username,
		Email:    a.Email,
		Phone:    a.Phone,
	}.Normalize()
	if err != nil {
		return err
	}
	if err := validateDisplayName(a.DisplayName); err != nil {
		return err
	}
	if err := a.Status.Validate(); err != nil {
		return err
	}
	if a.CreatedAt.IsZero() || a.UpdatedAt.IsZero() {
		return fmt.Errorf("account timestamps must not be zero")
	}
	if normalized.isEmpty() {
		return fmt.Errorf("at least one identity value is required")
	}

	return nil
}

func (c PasswordCredential) Validate() error {
	if _, err := normalizeAccountID(c.AccountID); err != nil {
		return err
	}
	if c.PasswordChangedAt.IsZero() || c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
		return fmt.Errorf("password credential timestamps must not be zero")
	}

	return nil
}

func (i Identity) Normalize() (NormalizedIdentity, error) {
	normalized := NormalizedIdentity{}

	if strings.TrimSpace(i.Username) != "" {
		username, usernameKey, err := normalizeUsername(i.Username)
		if err != nil {
			return NormalizedIdentity{}, err
		}
		normalized.Username = username
		normalized.UsernameKey = usernameKey
	}

	if strings.TrimSpace(i.Email) != "" {
		email, emailKey, err := normalizeEmail(i.Email)
		if err != nil {
			return NormalizedIdentity{}, err
		}
		normalized.Email = email
		normalized.EmailKey = emailKey
	}

	if strings.TrimSpace(i.Phone) != "" {
		phone, phoneKey, err := normalizePhone(i.Phone)
		if err != nil {
			return NormalizedIdentity{}, err
		}
		normalized.Phone = phone
		normalized.PhoneKey = phoneKey
	}

	if normalized.isEmpty() {
		return NormalizedIdentity{}, fmt.Errorf("at least one identity value is required")
	}

	return normalized, nil
}

func (i UpdateProfileInput) apply(current Account) (NormalizedIdentity, string, error) {
	nextIdentity := Identity{
		Username: current.Username,
		Email:    current.Email,
		Phone:    current.Phone,
	}

	if i.Username != nil {
		nextIdentity.Username = *i.Username
	}
	if i.Email != nil {
		nextIdentity.Email = *i.Email
	}
	if i.Phone != nil {
		nextIdentity.Phone = *i.Phone
	}

	displayName := current.DisplayName
	if i.DisplayName != nil {
		displayName = *i.DisplayName
	}

	normalized, err := nextIdentity.Normalize()
	if err != nil {
		return NormalizedIdentity{}, "", err
	}
	if err := validateDisplayName(displayName); err != nil {
		return NormalizedIdentity{}, "", err
	}

	return normalized, strings.TrimSpace(displayName), nil
}

func normalizeUsername(input string) (string, string, error) {
	username := strings.TrimSpace(input)
	if !usernamePattern.MatchString(username) {
		return "", "", fmt.Errorf("username must be 3-32 ASCII characters using letters, digits, dot, underscore, or hyphen")
	}

	return username, strings.ToLower(username), nil
}

func normalizeEmail(input string) (string, string, error) {
	email := strings.TrimSpace(input)
	if len(email) > EmailMaxLength {
		return "", "", fmt.Errorf("email must be at most 320 characters")
	}
	if !isASCII(email) {
		return "", "", fmt.Errorf("email must use ASCII characters only")
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || strings.Count(email, "@") != 1 {
		return "", "", fmt.Errorf("email must be a valid address")
	}

	return email, strings.ToLower(email), nil
}

func normalizePhone(input string) (string, string, error) {
	phone := strings.TrimSpace(input)
	if len(phone) > PhoneMaxLength {
		return "", "", fmt.Errorf("phone must be at most 16 characters")
	}
	if !phonePattern.MatchString(phone) {
		return "", "", fmt.Errorf("phone must use explicit E.164 format such as +14155550123")
	}

	return phone, phone, nil
}

func validateAccountID(input string) error {
	_, err := normalizeAccountID(input)
	return err
}

func normalizeAccountID(input string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return "", fmt.Errorf("account id must be a valid UUID")
	}

	return parsed.String(), nil
}

func validateDisplayName(input string) error {
	displayName := strings.TrimSpace(input)
	if displayName == "" {
		return fmt.Errorf("display name must not be empty")
	}
	if len([]rune(displayName)) > DisplayNameMaxLength {
		return fmt.Errorf("display name must be at most 120 characters")
	}

	return nil
}

func validatePasswordInput(password string) error {
	switch length := len(password); {
	case length == 0:
		return fmt.Errorf("password must not be empty")
	case length < PasswordMinBytes:
		return fmt.Errorf("password must be at least %d bytes", PasswordMinBytes)
	case length > PasswordMaxBytes:
		return fmt.Errorf("password must be at most %d bytes", PasswordMaxBytes)
	default:
		return nil
	}
}

func (n NormalizedIdentity) isEmpty() bool {
	return n.UsernameKey == "" && n.EmailKey == "" && n.PhoneKey == ""
}

func isASCII(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] > 0x7f {
			return false
		}
	}

	return true
}
