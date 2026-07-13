package message

import (
	"context"
	"fmt"
	"time"

	"github.com/nightzjp/kafka-manager/internal/cluster"
	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaBackend struct{cfg config.ClusterConfig; producer *kgo.Client}
func NewKafkaBackend(cfg config.ClusterConfig,producer *kgo.Client)*KafkaBackend{return &KafkaBackend{cfg:cfg,producer:producer}}

func(b *KafkaBackend)Fetch(ctx context.Context,q Query)([]Record,error){
	admin:=kadm.NewClient(b.producer);partitions:=[]int32{q.Partition};if q.Partition==-1{details,err:=admin.ListTopics(ctx,q.Topic);if err!=nil{return nil,err};detail,ok:=details[q.Topic];if !ok{return nil,fmt.Errorf("topic not found")};partitions=detail.Partitions.Numbers()}
	assignment:=map[string]map[int32]kgo.Offset{q.Topic:{}}
	for _,partition:=range partitions{offset,err:=b.resolveOffset(ctx,admin,q,partition);if err!=nil{return nil,err};assignment[q.Topic][partition]=kgo.NewOffset().At(offset)}
	opts,err:=cluster.Options(b.cfg);if err!=nil{return nil,err};opts=append(opts,kgo.ConsumePartitions(assignment),kgo.FetchMaxBytes(20*1024*1024))
	consumer,err:=kgo.NewClient(opts...);if err!=nil{return nil,err};defer consumer.Close()
	records:=make([]Record,0,q.Limit);for len(records)<q.Limit{fetchCtx,cancel:=context.WithTimeout(ctx,1200*time.Millisecond);batch:=consumer.PollRecords(fetchCtx,q.Limit-len(records));cancel();if errs:=batch.Errors();len(errs)>0{return nil,errs[0].Err};if batch.Empty(){break};batch.EachRecord(func(record *kgo.Record){headers:=make([]Header,0,len(record.Headers));for _,h:=range record.Headers{headers=append(headers,Header{Key:h.Key,Value:string(h.Value)})};records=append(records,Record{Topic:record.Topic,Partition:record.Partition,Offset:record.Offset,Timestamp:record.Timestamp.UnixMilli(),Key:string(record.Key),Value:string(record.Value),Headers:headers})})};return records,nil
}
func(b *KafkaBackend)resolveOffset(ctx context.Context,admin *kadm.Client,q Query,partition int32)(int64,error){if q.Mode=="offset"{return q.Offset,nil};var listed kadm.ListedOffsets;var err error;switch q.Mode{case"earliest":listed,err=admin.ListStartOffsets(ctx,q.Topic);case"latest":listed,err=admin.ListEndOffsets(ctx,q.Topic);case"timestamp":listed,err=admin.ListOffsetsAfterMilli(ctx,q.Timestamp,q.Topic)};if err!=nil{return 0,err};offset,ok:=listed.Lookup(q.Topic,partition);if !ok||offset.Err!=nil{return 0,fmt.Errorf("offset unavailable for partition %d",partition)};at:=offset.Offset;if q.Mode=="latest"{at-=int64(q.Limit);if at<0{at=0}};return at,nil}
func(b *KafkaBackend)Produce(ctx context.Context,r ProduceRequest)(Record,error){headers:=make([]kgo.RecordHeader,0,len(r.Headers));for _,h:=range r.Headers{headers=append(headers,kgo.RecordHeader{Key:h.Key,Value:[]byte(h.Value)})};record:=&kgo.Record{Topic:r.Topic,Key:[]byte(r.Key),Value:[]byte(r.Value),Headers:headers};if r.Partition>=0{record.Partition=r.Partition};result,err:=b.producer.ProduceSync(ctx,record).First();if err!=nil{return Record{},err};return Record{Topic:result.Topic,Partition:result.Partition,Offset:result.Offset,Timestamp:result.Timestamp.UnixMilli(),Key:r.Key,Value:r.Value,Headers:r.Headers},nil}
