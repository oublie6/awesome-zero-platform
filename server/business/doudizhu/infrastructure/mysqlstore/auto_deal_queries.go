package mysqlstore

import (
 "context"
 "fmt"

 "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
)

// PendingDeals is a bounded scan of durable hand phase state, not an in-memory
// queue. A crash or replacement worker can continue from the same database.
// Processing is idempotent and guarded by aggregate version and actor sequence.
func (s *Store) PendingDeals(ctx context.Context,limit int)([]domain.HandID,error){
 if s==nil || s.database==nil || s.database.DB()==nil{return nil,fmt.Errorf("hand store unavailable")}
 if limit<1 || limit>256{return nil,fmt.Errorf("invalid pending deal scan limit")}
 rows,err:=s.database.DB().QueryContext(ctx,`
SELECT hand_id FROM doudizhu_hands
WHERE phase IN ('FAIRNESS_COMMITTING','FAIRNESS_REVEALING','WAITING_PUBLIC_BEACON','DEALING')
ORDER BY
 CASE phase WHEN 'DEALING' THEN 0 WHEN 'WAITING_PUBLIC_BEACON' THEN 1 ELSE 2 END,
 updated_at ASC,hand_id ASC
LIMIT ?`,limit)
 if err!=nil{return nil,fmt.Errorf("scan pending fair deals: %w",err)}
 defer rows.Close()
 ids:=make([]domain.HandID,0,limit)
 for rows.Next(){
  var id domain.HandID
  if err:=rows.Scan(&id);err!=nil{return nil,err}
  ids=append(ids,id)
 }
 return ids,rows.Err()
}
