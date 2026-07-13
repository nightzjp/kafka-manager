package consumer

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeBackend struct { groups []Group }
func (f fakeBackend) Groups(context.Context) ([]Group,error){ return f.groups,nil }
func (f fakeBackend) Reset(context.Context,ResetRequest) error{return nil}
func (f fakeBackend) Delete(context.Context,string) error{return nil}

func TestListSortsGroupsByLag(t *testing.T){
	service:=NewService(fakeBackend{groups:[]Group{{Name:"small",TotalLag:2},{Name:"large",TotalLag:50}}})
	groups,err:=service.List(context.Background(),"")
	if err!=nil||len(groups)!=2||groups[0].Name!="large"{t.Fatalf("List() = %+v, %v",groups,err)}
}

func TestPartitionLagJSONUsesDistinctFields(t *testing.T){
	data,err:=json.Marshal(PartitionLag{Topic:"orders",Partition:1,CurrentOffset:2,EndOffset:5,Lag:3});if err!=nil{t.Fatal(err)}
	for _,field:=range []string{`"currentOffset":2`,`"endOffset":5`,`"lag":3`}{if !strings.Contains(string(data),field){t.Fatalf("json %s missing %s",data,field)}}
}

func TestResetRejectsUnsafeRequests(t *testing.T){
	service:=NewService(fakeBackend{})
	for _,request:=range []ResetRequest{{Group:"",Mode:"latest"},{Group:"orders",Mode:"unknown"},{Group:"orders",Mode:"absolute",Offset:-1}}{
		if err:=service.Reset(context.Background(),request);err==nil{t.Fatalf("accepted %+v",request)}
	}
}
