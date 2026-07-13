package consumer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/twmb/franz-go/pkg/kadm"
)

type PartitionLag struct {
	Topic string `json:"topic"`
	Partition int32 `json:"partition"`
	CurrentOffset int64 `json:"currentOffset"`
	EndOffset int64 `json:"endOffset"`
	Lag int64 `json:"lag"`
}
type Group struct { Name,State,Protocol string; MemberCount int; TotalLag int64; Partitions []PartitionLag }
type ResetRequest struct { Group string `json:"group"`; Mode string `json:"mode"`; Offset int64 `json:"offset,omitempty"`; Timestamp int64 `json:"timestamp,omitempty"`; Topics []string `json:"topics,omitempty"` }

type Backend interface { Groups(context.Context)([]Group,error); Reset(context.Context,ResetRequest)error; Delete(context.Context,string)error }
type Service struct{backend Backend}
func NewService(backend Backend)*Service{return &Service{backend:backend}}
func(s *Service)List(ctx context.Context,search string)([]Group,error){groups,err:=s.backend.Groups(ctx);if err!=nil{return nil,err};search=strings.ToLower(strings.TrimSpace(search));out:=groups[:0];for _,g:=range groups{if search==""||strings.Contains(strings.ToLower(g.Name),search){out=append(out,g)}};sort.Slice(out,func(i,j int)bool{if out[i].TotalLag==out[j].TotalLag{return out[i].Name<out[j].Name};return out[i].TotalLag>out[j].TotalLag});return out,nil}
func(s *Service)Reset(ctx context.Context,r ResetRequest)error{if strings.TrimSpace(r.Group)==""{return fmt.Errorf("group is required")};switch r.Mode{case"earliest","latest":case"absolute":if r.Offset<0{return fmt.Errorf("offset must not be negative")};case"timestamp":if r.Timestamp<=0{return fmt.Errorf("timestamp is required")};default:return fmt.Errorf("unsupported reset mode %q",r.Mode)};return s.backend.Reset(ctx,r)}
func(s *Service)Delete(ctx context.Context,group string)error{if strings.TrimSpace(group)==""{return fmt.Errorf("group is required")};return s.backend.Delete(ctx,group)}

type KadmBackend struct{client *kadm.Client}
func NewKadmBackend(client *kadm.Client)*KadmBackend{return &KadmBackend{client:client}}
func(b *KadmBackend)Groups(ctx context.Context)([]Group,error){listed,err:=b.client.ListGroups(ctx);if err!=nil{return nil,err};names:=listed.Groups();if len(names)==0{return []Group{},nil};lags,err:=b.client.Lag(ctx,names...);if err!=nil{return nil,err};groups:=make([]Group,0,len(lags));for _,l:=range lags.Sorted(){if l.Error()!=nil{continue};g:=Group{Name:l.Group,State:l.State,Protocol:l.Protocol,MemberCount:len(l.Members),TotalLag:l.Lag.Total()};for _,p:=range l.Lag.Sorted(){g.Partitions=append(g.Partitions,PartitionLag{Topic:p.Topic,Partition:p.Partition,CurrentOffset:p.Commit.At,EndOffset:p.End.Offset,Lag:p.Lag})};groups=append(groups,g)};return groups,nil}
func(b *KadmBackend)Reset(ctx context.Context,r ResetRequest)error{lags,err:=b.client.Lag(ctx,r.Group);if err!=nil{return err};lag,ok:=lags[r.Group];if !ok{return fmt.Errorf("consumer group not found")};if lag.State!="Empty"&&lag.State!="Dead"{return fmt.Errorf("consumer group must be stopped before resetting offsets")};topics:=r.Topics;if len(topics)==0{seen:=map[string]bool{};for _,p:=range lag.Lag.Sorted(){if !seen[p.Topic]{topics=append(topics,p.Topic);seen[p.Topic]=true}}};var offsets kadm.Offsets;switch r.Mode{case"earliest":listed,e:=b.client.ListStartOffsets(ctx,topics...);if e!=nil{return e};offsets=listed.Offsets();case"latest":listed,e:=b.client.ListEndOffsets(ctx,topics...);if e!=nil{return e};offsets=listed.Offsets();case"timestamp":listed,e:=b.client.ListOffsetsAfterMilli(ctx,r.Timestamp,topics...);if e!=nil{return e};offsets=listed.Offsets();case"absolute":offsets=make(kadm.Offsets);for _,p:=range lag.Lag.Sorted(){if len(r.Topics)>0&&!contains(r.Topics,p.Topic){continue};offsets.Add(kadm.Offset{Topic:p.Topic,Partition:p.Partition,At:r.Offset})}};return b.client.CommitAllOffsets(ctx,r.Group,offsets)}
func(b *KadmBackend)Delete(ctx context.Context,group string)error{_,err:=b.client.DeleteGroup(ctx,group);return err}
func contains(values []string,want string)bool{for _,v:=range values{if v==want{return true}};return false}
