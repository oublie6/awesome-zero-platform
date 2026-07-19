package requestid

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"strings"
)

type contextKey string

const key contextKey = "request-id"

var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func IntoContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, key, requestID)
}

func FromContext(ctx context.Context) string {
	value, _ := ctx.Value(key).(string)
	return value
}

func Effective(value string, maxLength int) string {
	if IsValid(value, maxLength) {
		return strings.TrimSpace(value)
	}

	return Generate()
}

func IsValid(value string, maxLength int) bool {
	value = strings.TrimSpace(value)
	if value == "" || maxLength <= 0 || len(value) > maxLength {
		return false
	}

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}

	return true
}

func Generate() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err)
	}

	return encoding.EncodeToString(raw[:])
}
