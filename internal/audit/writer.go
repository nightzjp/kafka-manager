package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct{Directory string;MaxFileSizeBytes int64;RetentionDays int}
type Entry struct{Timestamp time.Time `json:"timestamp"`;Username string `json:"username"`;ClientIP string `json:"clientIp,omitempty"`;ClusterID string `json:"clusterId,omitempty"`;Action string `json:"action"`;Resource string `json:"resource,omitempty"`;Result string `json:"result"`;DurationMS int64 `json:"durationMs,omitempty"`;Error string `json:"error,omitempty"`;Parameters map[string]any `json:"parameters,omitempty"`}
type Writer struct{mu sync.Mutex;cfg Config;now func()time.Time;date string;index int;size int64;file *os.File;buffer *bufio.Writer}
func NewWriter(cfg Config)(*Writer,error){if cfg.Directory==""{return nil,fmt.Errorf("audit directory is required")};if cfg.MaxFileSizeBytes<1{cfg.MaxFileSizeBytes=50*1024*1024};if cfg.RetentionDays<1{cfg.RetentionDays=30};return &Writer{cfg:cfg,now:time.Now},nil}
func(w *Writer)Write(_ context.Context,entry Entry)error{w.mu.Lock();defer w.mu.Unlock();now:=w.now();if entry.Timestamp.IsZero(){entry.Timestamp=now};data,err:=json.Marshal(entry);if err!=nil{return err};data=append(data,'\n');if err:=w.ensureFile(now,int64(len(data)));err!=nil{return err};n,err:=w.buffer.Write(data);w.size+=int64(n);if err!=nil{return err};return w.buffer.Flush()}
func(w *Writer)ensureFile(now time.Time,incoming int64)error{date:=now.Format("2006-01-02");if w.file!=nil&&w.date==date&&w.size+incoming<=w.cfg.MaxFileSizeBytes{return nil};if err:=w.closeLocked();err!=nil{return err};if w.date!=date{w.date=date;w.index=0};dir:=filepath.Join(w.cfg.Directory,date);if err:=os.MkdirAll(dir,0o700);err!=nil{return err};for{w.index++;path:=filepath.Join(dir,fmt.Sprintf("audit-%03d.jsonl",w.index));info,err:=os.Stat(path);if os.IsNotExist(err){file,e:=os.OpenFile(path,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0o600);if e!=nil{return e};w.file=file;w.buffer=bufio.NewWriter(file);w.size=0;return nil};if err!=nil{return err};if info.Size()+incoming<=w.cfg.MaxFileSizeBytes{file,e:=os.OpenFile(path,os.O_WRONLY|os.O_APPEND,0o600);if e!=nil{return e};w.file=file;w.buffer=bufio.NewWriter(file);w.size=info.Size();return nil}}}
func(w *Writer)Close()error{w.mu.Lock();defer w.mu.Unlock();return w.closeLocked()}
func(w *Writer)closeLocked()error{if w.file==nil{return nil};if err:=w.buffer.Flush();err!=nil{return err};err:=w.file.Close();w.file=nil;w.buffer=nil;return err}
