package bootstrap

import (
    "context"
    "database/sql"
    "errors"

    "github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/doudizhuapi"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
)
type turnClockAdapter struct{journal *mysqlstore.LiveJournal}
func(a turnClockAdapter)CurrentTurnClock(ctx context.Context,id domain.HandID)(doudizhuapi.TurnClock,error){
    record,now,err:=a.journal.GetTurnDeadline(ctx,id)
    if errors.Is(err,sql.ErrNoRows){return doudizhuapi.TurnClock{},nil}
    if err!=nil{return doudizhuapi.TurnClock{},err}
    return doudizhuapi.TurnClock{LiveVersion:record.Version,
        DeadlineMillis:record.Deadline.UnixMilli(),
        ServerTimeMillis:now.UnixMilli()},nil
}
