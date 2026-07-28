package textnormalization

import "testing"

func TestNFKCNormalizesCompatibilityCharacters(t *testing.T) {
	value, err := (NFKC{}).Normalize("原始短语Ａ①")
	if err != nil {
		t.Fatal(err)
	}
	if value != "原始短语A1" {
		t.Fatalf("normalized=%q", value)
	}
}
