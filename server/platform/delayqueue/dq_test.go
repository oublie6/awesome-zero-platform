package delayqueue

import "testing"

func TestRequiresTwoDistinctServers(t *testing.T){
    c:=Config{Endpoints:[]string{"beanstalk-a:11300","beanstalk-b:11300"},Tube:"doudizhu-turn",RedisHost:"redis:6379"}
    if err:=c.Validate();err!=nil{t.Fatal(err)}
    c.Endpoints=[]string{"beanstalk-a:11300"};if c.Validate()==nil{t.Fatal("single node incorrectly accepted")}
    c.Endpoints=[]string{"beanstalk-a:11300","beanstalk-a:11300"};if c.Validate()==nil{t.Fatal("duplicate node incorrectly accepted")}
    c.Endpoints=[]string{"beanstalk-a:11300","beanstalk-b:11300"}
    c.RedisHost="";if c.Validate()==nil{t.Fatal("missing dedup Redis accepted")}
}
