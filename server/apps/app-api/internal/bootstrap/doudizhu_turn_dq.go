package bootstrap

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/lifecycle"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/application/turntimer"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/bidding"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/carddeck"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/domain/livehand"
    "github.com/oublie6/awesome-zero-platform/server/business/doudizhu/infrastructure/mysqlstore"
    "github.com/oublie6/awesome-zero-platform/server/platform/delayqueue"
)

type turnDqPublisher interface {
    At(context.Context,turntimer.Ticket)(string,error)
    Close()error
}
type turnDqStore interface {
    GetTurnDeadline(context.Context,domain.HandID)(mysqlstore.TurnDeadline,time.Time,error)
    PendingTurnDeliveries(context.Context,int)([]mysqlstore.TurnDeadline,error)
    MarkTurnDelivery(context.Context,mysqlstore.TurnDeadline,bool)error
}
type turnDqReader interface{
    LoadHand(context.Context,domain.HandID)(domain.HandSnapshot,error)
}
type turnDqGame interface{
    GetHandPrivateView(context.Context,domain.AccountID,domain.HandID)(application.LiveHandView,error)
    SubmitLiveHandBidWithID(context.Context,domain.AccountID,domain.HandID,string,uint64,bidding.Score)(application.LiveHandCommandResult,error)
    SubmitLiveHandPlayWithID(context.Context,domain.AccountID,domain.HandID,string,uint64,[]string)(application.LiveHandCommandResult,error)
    SubmitLiveHandPassWithID(context.Context,domain.AccountID,domain.HandID,string,uint64)(application.LiveHandCommandResult,error)
}
type turnDqLifecycle interface {
    AfterBid(context.Context,domain.AccountID,domain.HandID,application.LiveHandCommandResult)(lifecycle.Outcome,error)
    AfterPlay(context.Context,domain.AccountID,domain.HandID,application.LiveHandCommandResult)error
}
type turnDqWorker struct{
    store turnDqStore
    reader turnDqReader
    game turnDqGame
    lifecycle turnDqLifecycle
    ids application.IDGenerator
    publisher turnDqPublisher
}

func verifyRecoveredTurnDeadlines(ctx context.Context,journal *mysqlstore.LiveJournal,states map[domain.HandID]string)error{
    for id,phase:=range states{
        if phase!=livehand.PhaseBidding && phase!=livehand.PhasePlaying{continue}
        entry,_,err:=journal.GetTurnDeadline(ctx,id)
        if err!=nil{return fmt.Errorf("active hand lacks stored turn deadline: %w",err)}
        if entry.Phase!=phase||entry.Version==0||entry.Seat<1||entry.Seat>3{
            return fmt.Errorf("active hand has incompatible persisted turn deadline")
        }
    }
    return nil
}
func startDoudizhuTurnDQ(ctx context.Context,cfg config.Config,
    reader turnDqReader,journal *mysqlstore.LiveJournal,game turnDqGame,
    lifecycleService turnDqLifecycle,ids application.IDGenerator)error{
    endpoints:=cfg.Doudizhu.TurnDQ.Endpoints
    queueConfig:=delayqueue.Config{Endpoints:endpoints,Tube:cfg.Doudizhu.TurnDQ.Tube,
        RedisHost:cfg.Redis.Addr,RedisUser:cfg.Redis.Username,RedisPass:cfg.Redis.Password}
    producer,err:=delayqueue.NewPublisher(queueConfig)
    if err!=nil{return err}
    consumer,err:=delayqueue.NewConsumer(queueConfig)
    if err!=nil{_ = producer.Close();return err}
    worker:=&turnDqWorker{store:journal,reader:reader,game:game,lifecycle:lifecycleService,ids:ids,publisher:producer}
    go func(){
        defer producer.Close()
        ticker:=time.NewTicker(cfg.Doudizhu.SweepInterval)
        defer ticker.Stop()
        for{
            if err:=ctx.Err();err!=nil{return}
            checkCtx,cancel:=context.WithTimeout(ctx,10*time.Second)
            err:=worker.dispatch(checkCtx,64)
            cancel()
            if err!=nil&&ctx.Err()==nil{log.Printf("turn DQ delivery repair error: %v",err)}
            select{case <-ctx.Done():return;case <-ticker.C:}
        }
    }()
    go consumer.Consume(func(ticket turntimer.Ticket)error{
        if ctx.Err()!=nil{return nil}
        handleCtx,cancel:=context.WithTimeout(ctx,10*time.Second)
        defer cancel()
        return worker.process(handleCtx,ticket)
    },func(err error){
        if ctx.Err()==nil{log.Printf("turn DQ consume callback failed; durable retry is required: %v",err)}
    })
    return nil
}

// The cursor is a *repair mechanism*, not a permanent per-hand timer. After
// dispatching, it reschedules a bounded retry at deadline+10s because dq may
// delete jobs before callbacks. New committed versions replace the row.
func(w *turnDqWorker)dispatch(ctx context.Context,limit int)error{
    items,err:=w.store.PendingTurnDeliveries(ctx,limit)
    if err!=nil{return err}
    for _,item:=range items{
        if err:=ctx.Err();err!=nil{return err}
        deliveryID,err:=w.ids.NewID()
        if err!=nil{return err}
        ticket:=turntimer.Ticket{HandID:string(item.HandID),LiveVersion:item.Version,
            Seat:item.Seat,DeadlineMillis:item.Deadline.UnixMilli(),DeliveryID:deliveryID}
        _,publishErr:=w.publisher.At(ctx,ticket)
        markErr:=w.store.MarkTurnDelivery(ctx,item,publishErr==nil)
        if markErr!=nil{return markErr}
        if publishErr!=nil{return publishErr}
    }
    return nil
}
func(w *turnDqWorker)process(ctx context.Context,ticket turntimer.Ticket)error{
    if err:=ticket.Validate();err!=nil{return err}
    handID:=domain.HandID(ticket.HandID)
    current,now,err:=w.store.GetTurnDeadline(ctx,handID)
    if errors.Is(err,sql.ErrNoRows){return nil} // terminal hand already archived
    if err!=nil{return err}
    if current.Version!=ticket.LiveVersion||current.Seat!=ticket.Seat||
        current.Deadline.UnixMilli()!=ticket.DeadlineMillis{return nil}
    if now.Before(current.Deadline){return nil} // never execute prematurely
    snapshot,err:=w.reader.LoadHand(ctx,handID)
    if err!=nil{return err}
    if snapshot.Phase.Terminal(){return nil}
    var actor domain.AccountID
    for _,seat:=range snapshot.Seats{
        if uint8(seat.Seat)==current.Seat{actor=seat.AccountID;break}
    }
    if actor==""{return fmt.Errorf("turn DQ seat has no committed participant")}
    private,err:=w.game.GetHandPrivateView(ctx,actor,handID)
    if err!=nil{return err}
    var view livehand.PrivateView
    if err:=json.Unmarshal(private.Payload,&view);err!=nil{return err}
    if private.Version!=current.Version||view.Public.StateVersion!=current.Version||
       view.Public.Phase!=current.Phase ||view.Position!=current.Seat {
        // Stale callback or an in-flight user action won the version race.
        return nil
    }
    commandID:=turnTimeoutCommandID(current)
    var result application.LiveHandCommandResult
    switch current.Phase{
    case livehand.PhaseBidding:
        if view.Public.Bidding.CurrentBidder!=current.Seat{return nil}
        result,err=w.game.SubmitLiveHandBidWithID(ctx,actor,handID,commandID,current.Version,bidding.ScorePass)
        if err==nil{_,err=w.lifecycle.AfterBid(ctx,actor,handID,result)}
    case livehand.PhasePlaying:
        if view.Public.Playing==nil||view.Public.Playing.CurrentSeat!=current.Seat{return nil}
        if len(view.Public.Playing.LeadingCards)>0{
            result,err=w.game.SubmitLiveHandPassWithID(ctx,actor,handID,commandID,current.Version)
        }else{
            card,selectErr:=lowestSingleCard(view.Cards)
            if selectErr!=nil{return selectErr}
            result,err=w.game.SubmitLiveHandPlayWithID(ctx,actor,handID,commandID,current.Version,[]string{card})
        }
        if err==nil{err=w.lifecycle.AfterPlay(ctx,actor,handID,result)}
    default:return fmt.Errorf("unsupported persisted turn phase")
    }
    if err==nil{return nil}
    // If a concurrent action advanced the version, this timeout is obsolete,
    // but any remaining failure must be visible to the durable repair scanner.
    latest,_,lookupErr:=w.store.GetTurnDeadline(ctx,handID)
    if errors.Is(lookupErr,sql.ErrNoRows){return nil}
    if lookupErr==nil && latest.Version!=current.Version{return nil}
    return err
}
// A deterministic, actor-bound ID makes unknown-result retries idempotent.
// Client calls always have a nonzero client sequence; the journal recognizes
// this prefix only for internal zero-sequence commands.
func turnTimeoutCommandID(t mysqlstore.TurnDeadline)string{
    digest:=sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d",t.HandID,t.Version,t.Seat)))
    return "turn-timeout:"+hex.EncodeToString(digest[:])
}
func lowestSingleCard(cards []string)(string,error){
    var chosen string
    best:=carddeck.Card(255)
    for _,code:=range cards{
        card,err:=carddeck.ParseCard(code)
        if err!=nil{return "",err}
        if card<best{best=card;chosen=code}
    }
    if chosen==""{return "",errors.New("no private card to lead")}
    return chosen,nil
}
