package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriterCreatesDailyDirectoryAndRotates(t *testing.T){dir:=t.TempDir();now:=time.Date(2026,7,13,10,0,0,0,time.Local);writer,err:=NewWriter(Config{Directory:dir,MaxFileSizeBytes:120,RetentionDays:30});if err!=nil{t.Fatal(err)};writer.now=func()time.Time{return now};defer writer.Close();for i:=0;i<4;i++{if err:=writer.Write(context.Background(),Entry{Username:"admin",ClusterID:"dev",Action:"topic.create",Resource:"orders",Result:"success"});err!=nil{t.Fatal(err)}};files,err:=filepath.Glob(filepath.Join(dir,"2026-07-13","audit-*.jsonl"));if err!=nil||len(files)<2{t.Fatalf("files=%v err=%v",files,err)}}
func TestCleanupRemovesExpiredDirectories(t *testing.T){dir:=t.TempDir();old:=filepath.Join(dir,"2026-06-01");recent:=filepath.Join(dir,"2026-07-12");if err:=os.MkdirAll(old,0o700);err!=nil{t.Fatal(err)};if err:=os.MkdirAll(recent,0o700);err!=nil{t.Fatal(err)};if err:=Cleanup(dir,30,time.Date(2026,7,13,0,0,0,0,time.Local));err!=nil{t.Fatal(err)};if _,err:=os.Stat(old);!os.IsNotExist(err){t.Fatal("expired directory remains")};if _,err:=os.Stat(recent);err!=nil{t.Fatal("recent directory removed")}}
func TestQueryFiltersEntries(t *testing.T){dir:=t.TempDir();writer,err:=NewWriter(Config{Directory:dir,MaxFileSizeBytes:1024,RetentionDays:30});if err!=nil{t.Fatal(err)};writer.now=func()time.Time{return time.Date(2026,7,13,10,0,0,0,time.Local)};_ = writer.Write(context.Background(),Entry{ClusterID:"dev",Action:"topic.create",Result:"success"});_ = writer.Write(context.Background(),Entry{ClusterID:"test",Action:"topic.delete",Result:"failed"});_ = writer.Close();items,err:=Query(dir,Filter{From:"2026-07-13",To:"2026-07-13",ClusterID:"test",Limit:20});if err!=nil||len(items)!=1||items[0].Action!="topic.delete"{t.Fatalf("Query=%+v,%v",items,err)}}
