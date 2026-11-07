package mysqlstore

import (
 "context"
 "fmt"
 "time"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// LoadProtectedContribution only reads a record bound to the caller-authorized
// hand, seat and account. Application policy additionally checks terminal
// status before allowing any plaintext-derived response.
func (s *Store) LoadProtectedContribution(ctx context.Context, recordID string,
 handID domain.HandID, seat domain.Seat, actor domain.AccountID,
) (application.ProtectedContributionRecord,error) {
 if s==nil || s.database==nil || recordID=="" || handID=="" || !seat.Valid() || actor==""{
  return application.ProtectedContributionRecord{},fmt.Errorf("invalid protected contribution identity")
 }
 var record application.ProtectedContributionRecord
 var digest,aad []byte
 var seatNumber uint8
 err:=s.database.DB().QueryRowContext(ctx,`
SELECT record_id, hand_id, seat_number, actor_account_id, command_id,
       contribution_digest, protection_key_id, nonce, ciphertext, aad_digest, created_at
FROM doudizhu_contribution_records
WHERE record_id=? AND hand_id=? AND seat_number=? AND actor_account_id=?`,
  recordID,handID,seat,actor).Scan(
  &record.RecordID,&record.HandID,&seatNumber,&record.ActorAccountID,&record.CommandID,
  &digest,&record.ProtectionKeyID,&record.Nonce,&record.Ciphertext,&aad,&record.CreatedAt)
 if err!=nil{return application.ProtectedContributionRecord{},err}
 if len(digest)!=32 || len(aad)!=32 || seatNumber!=uint8(seat) ||
    record.RecordID!=recordID || record.HandID!=handID || record.ActorAccountID!=actor ||
    record.CommandID=="" || record.ProtectionKeyID=="" ||
    len(record.Nonce)==0 || len(record.Ciphertext)==0 {
  return application.ProtectedContributionRecord{},fmt.Errorf("invalid bound protected contribution row")
 }
 record.Seat=seat
 copy(record.ContributionDigest[:],digest)
 copy(record.AADDigest[:],aad)
 record.CreatedAt=record.CreatedAt.UTC().Truncate(time.Microsecond)
 return record,nil
}
