package carddeck_test

import (
	"testing"

	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
	"github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
)

func TestClientCommitmentMatchesExistingDomainContract(t *testing.T) {
	handID := "hand-commitment-compatibility"
	for seat := uint8(1); seat <= 3; seat++ {
		var contribution carddeck.ContributionDigest
		for index := range contribution {
			contribution[index] = byte(int(seat)*31 + index)
		}
		got, err := carddeck.ComputeClientCommitment(handID, seat, contribution)
		if err != nil {
			t.Fatal(err)
		}
		var domainContribution domain.ContributionDigest
		copy(domainContribution[:], contribution[:])
		want := domain.ComputeClientCommitment(domain.HandID(handID), domain.Seat(seat), domainContribution)
		if got != carddeck.Commitment(want) {
			t.Fatalf("seat=%d got=%x want=%x", seat, got, want)
		}
	}
}
