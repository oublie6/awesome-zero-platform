package textnormalization

import (
	"fmt"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

type NFKC struct{}

func (NFKC) Normalize(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("phrase is not valid UTF-8")
	}
	return norm.NFKC.String(value), nil
}
