package delayqueue

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/zeromicro/go-queue/dq"
    zeroredis "github.com/zeromicro/go-zero/core/stores/redis"
)

// Config isolates infrastructure; one dq.Producer requires 2 distinct nodes.
// Redis DB 0 is used by go-queue solely for its deduplication lock.
type Config struct {
    Endpoints []string
    Tube string
    RedisHost string
    RedisUser string
    RedisPass string
}
func (c Config) Validate()error{
    if len(c.Endpoints)<2 || len(c.Endpoints)>3 || strings.TrimSpace(c.Tube)=="" ||
        strings.TrimSpace(c.RedisHost)=="" {return errors.New("dq requires two distinct beanstalkd nodes, tube and redis")}
    distinct:=make(map[string]struct{},len(c.Endpoints))
    for _,ep:=range c.Endpoints{
        if strings.TrimSpace(ep)!=ep || ep=="" {return errors.New("invalid beanstalkd endpoint")}
        if _,ok:=distinct[ep];ok{return errors.New("duplicate beanstalkd endpoint")}
        distinct[ep]=struct{}{}
    }
    return nil
}
func (c Config) beanstalks()[]dq.Beanstalk{
    list:=make([]dq.Beanstalk,0,len(c.Endpoints))
    for _,ep:=range c.Endpoints {list=append(list,dq.Beanstalk{Endpoint:ep,Tube:c.Tube})}
    return list
}
type Publisher struct{ inner dq.Producer }
func NewPublisher(c Config)(*Publisher,error){
    if err:=c.Validate();err!=nil{return nil,err}
    return &Publisher{inner:dq.NewProducer(c.beanstalks())},nil
}
// At may return an error even if a minority of replicas accepted the job;
// all deliveries must therefore be idempotent and reconciled from MySQL.
func(p *Publisher)At(ctx context.Context,body []byte, deadline time.Time)(string,error){
    if p==nil||p.inner==nil{return "",errors.New("dq producer unavailable")}
    if err:=ctx.Err();err!=nil{return "",err}
    if len(body)==0 || len(body)>1024{return "",errors.New("invalid delay queue message length")}
    at:=deadline
    if !at.After(time.Now()) {at=time.Now().Add(time.Second)}
    id,err:=p.inner.At(body,at)
    if err!=nil{return "",fmt.Errorf("schedule turn timeout: %w",err)}
    return id,nil
}
func(p *Publisher)Close()error{
    if p==nil||p.inner==nil{return nil}
    return p.inner.Close()
}
type Consumer struct{inner dq.Consumer}
func NewConsumer(c Config)(*Consumer,error){
    if err:=c.Validate();err!=nil{return nil,err}
    red:=zeroredis.RedisConf{Host:c.RedisHost,Type:zeroredis.NodeType,User:c.RedisUser,Pass:c.RedisPass}
    return &Consumer{inner:dq.NewConsumer(dq.DqConf{Beanstalks:c.beanstalks(),Redis:red})},nil
}
// Consume blocks. The library deletes beanstalkd jobs BEFORE calling handler
// and Redis deduplicates bodies for one hour. The caller MUST reconcile failed
// and missing jobs from the durable MySQL schedule; a callback alone is unsafe.
func(c *Consumer)Consume(handler func([]byte)error,onError func(error)){
    if c==nil||c.inner==nil{if onError!=nil{onError(errors.New("dq consumer unavailable"))};return}
    c.inner.Consume(func(body []byte){
        err:=handler(body)
        if err!=nil&&onError!=nil{onError(err)}
    })
}
