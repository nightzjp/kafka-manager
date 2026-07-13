import { FormEvent, useEffect, useRef, useState } from 'react';
import { Empty, ErrorNotice, PageHeader, Status } from '../components/Common';
import { Icon } from '../components/Icon';
import { JsonViewer, parseMessageValue } from '../components/JsonViewer';
import { api } from '../lib/api';
import { MessageRecord } from '../lib/types';

export function MessagesPage({ clusterId, fixedTopic, embedded = false }: { clusterId: string; fixedTopic?: string; embedded?: boolean }) {
  const [items, setItems] = useState<MessageRecord[]>([]);
  const [selected, setSelected] = useState<MessageRecord>();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [producer, setProducer] = useState(false);
  const [live, setLive] = useState(false);
  const [topic, setTopic] = useState(fixedTopic || '');
  const [partition, setPartition] = useState(-1);
  const [mode, setMode] = useState('latest');
  const [offset, setOffset] = useState(0);
  const [timestamp, setTimestamp] = useState('');
  const [limit, setLimit] = useState(100);
  const stream = useRef<EventSource | null>(null);

  const stop = () => { stream.current?.close(); stream.current = null; setLive(false); };
  async function queryMessages(requestedTopic = topic) {
    stop(); setLoading(true);
    try {
      const query = new URLSearchParams({ topic: requestedTopic, partition: String(partition), mode, limit: String(limit), offset: String(offset), timestamp: String(timestamp ? new Date(timestamp).getTime() : 0) });
      const result = await api.get<{ items: MessageRecord[] }>(`/api/v1/clusters/${clusterId}/messages?${query}`);
      setItems(result.items); setSelected(result.items[0]); setError('');
    } catch (reason) { setError(reason instanceof Error ? reason.message : '查询失败'); }
    finally { setLoading(false); }
  }

  useEffect(() => {
    if (!fixedTopic) { stop(); return; }
    setTopic(fixedTopic);
    void queryMessages(fixedTopic);
    return stop;
    // A fixed Topic automatically loads once; subsequent filter changes are explicit queries.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, fixedTopic]);
  useEffect(() => () => stream.current?.close(), []);

  function follow() {
    if (live) { stop(); return; }
    if (!topic.trim()) { setError('请先选择 Topic'); return; }
    setItems([]); setSelected(undefined); setError('');
    const query = new URLSearchParams({ topic, partition: String(partition) });
    const source = new EventSource(`/api/v1/clusters/${clusterId}/messages/stream?${query}`);
    stream.current = source;
    source.onmessage = (event) => { const record = JSON.parse(event.data) as MessageRecord; setItems((current) => [record, ...current].slice(0, 500)); };
    source.onerror = () => { setError('实时连接已中断，可重新开始跟随'); stop(); };
    setLive(true);
  }

  return <div className={embedded ? 'messages-embedded' : ''}>
    {!embedded && <PageHeader code="MESSAGES" title="消息检索" description="跨 Topic 查询 Kafka 原始消息；也可以从 Topic 工作区直接进入。" actions={<button className="button primary" onClick={() => setProducer(!producer)}><Icon name="plus" />生产消息</button>} />}
    {embedded && <div className="panel-heading"><div><h2>消息</h2><p>当前 Topic 已自动绑定，并载入最近消息</p></div><button className="button primary" onClick={() => setProducer(!producer)}><Icon name="plus" />生产消息</button></div>}
    <form className="message-query" onSubmit={(event) => { event.preventDefault(); void queryMessages(); }}>
      {!fixedTopic && <label className="wide">Topic<input required value={topic} onChange={(event) => setTopic(event.target.value)} placeholder="输入 Topic 名称" /></label>}
      <label>Partition<input type="number" min="-1" value={partition} onChange={(event) => setPartition(Number(event.target.value))} /></label>
      <label>起始位置<select value={mode} onChange={(event) => setMode(event.target.value)}><option value="latest">最近消息</option><option value="earliest">最早消息</option><option value="offset">指定 Offset</option><option value="timestamp">指定时间</option></select></label>
      {mode === 'offset' && <label>Offset<input type="number" min="0" value={offset} onChange={(event) => setOffset(Number(event.target.value))} /></label>}
      {mode === 'timestamp' && <label>时间<input type="datetime-local" required value={timestamp} onChange={(event) => setTimestamp(event.target.value)} /></label>}
      <label>数量<input type="number" min="1" max="500" value={limit} onChange={(event) => setLimit(Number(event.target.value))} /></label>
      <div className="query-actions"><button className="button primary" disabled={loading}><Icon name="search" />{loading ? '查询中' : '查询'}</button><button type="button" className={`button ${live ? 'danger' : 'ghost'}`} onClick={follow}><Icon name={live ? 'stop' : 'play'} />{live ? '停止' : '实时跟随'}</button></div>
    </form>
    {live && <div className="live-strip"><span />正在监听新消息，仅保留最近 500 条</div>}
    {error && <ErrorNotice message={error} />}
    {producer && <Producer clusterId={clusterId} fixedTopic={fixedTopic} done={() => setProducer(false)} />}
    <div className="message-browser">
      <div className="message-list">{items.length === 0 ? <Empty title={live ? '等待新消息' : loading ? '正在载入' : '没有消息'} detail={live ? '连接已建立，正在监听新记录。' : loading ? '正在读取所选 Topic 的消息。' : '当前查询条件没有返回记录。'} /> : items.map((record) => {
        const parsed = parseMessageValue(record.value);
        return <button key={`${record.partition}-${record.offset}`} className={selected === record ? 'active' : ''} onClick={() => setSelected(record)}><div><Status tone="neutral">P{record.partition}</Status><span className={`type-badge ${parsed.kind}`}>{parsed.kind.toUpperCase()}</span><time>{new Date(record.timestamp).toLocaleString()}</time></div><strong>Offset {record.offset}</strong><code>{record.key || '无 Key'}</code><p>{preview(record.value)}</p></button>;
      })}</div>
      <div className="message-detail">{selected ? <><header><div><span>Partition {selected.partition}</span><b>Offset {selected.offset}</b></div><time>{new Date(selected.timestamp).toLocaleString()}</time></header><section className="record-meta"><div><span>Key</span><code>{selected.key || 'null'}</code></div><div><span>Headers</span><code>{selected.headers?.length || 0} 项</code></div></section><JsonViewer raw={selected.value} />{selected.headers?.length > 0 && <section className="headers"><h3>Headers</h3><pre>{JSON.stringify(selected.headers, null, 2)}</pre></section>}</> : <div className="detail-placeholder">从左侧选择一条消息查看内容</div>}</div>
    </div>
  </div>;
}

function preview(value: string) { const parsed = parseMessageValue(value); const text = parsed.kind === 'json' ? JSON.stringify(parsed.value) : value; return text.length > 120 ? `${text.slice(0, 120)}…` : text; }

function Producer({ clusterId, fixedTopic, done }: { clusterId: string; fixedTopic?: string; done: () => void }) {
  const [error, setError] = useState(''); const [success, setSuccess] = useState(''); const [sending, setSending] = useState(false);
  const [kind, setKind] = useState<'json' | 'text'>('json'); const [value, setValue] = useState('{\n  \n}');
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (sending) return;
    if (kind === 'json') { try { JSON.parse(value); } catch (reason) { setError(`JSON 格式错误：${reason instanceof Error ? reason.message : '无法解析'}`); return; } }
    const data = new FormData(event.currentTarget); setSending(true); setError(''); setSuccess('');
    try { const record = await api.post<MessageRecord>(`/api/v1/clusters/${clusterId}/messages`, { topic: fixedTopic || data.get('topic'), partition: Number(data.get('partition')), key: data.get('key'), value, headers: [] }); setSuccess(`消息已发送到 Partition ${record.partition}，Offset ${record.offset}`); }
    catch (reason) { setError(reason instanceof Error ? reason.message : '发送失败'); }
    finally { setSending(false); }
  }
  return <form className="producer-card" onSubmit={submit}><div className="panel-heading"><div><h2>生产消息</h2><p>消息正文不会写入审计日志</p></div><div className="segmented"><button type="button" className={kind === 'json' ? 'active' : ''} onClick={() => setKind('json')}>JSON</button><button type="button" className={kind === 'text' ? 'active' : ''} onClick={() => setKind('text')}>Text</button></div></div><div className="producer-fields">{!fixedTopic && <label>Topic<input name="topic" required /></label>}<label>Partition<input name="partition" type="number" defaultValue="-1" /></label><label>Key<input name="key" placeholder="可选" /></label></div><label>Value<textarea className="code-editor" value={value} onChange={(event) => setValue(event.target.value)} required spellCheck={false} /></label>{error && <ErrorNotice message={error} />}{success && <div className="success-notice" role="status"><Icon name="check" />{success}</div>}<div className="form-actions"><button type="button" className="button ghost" onClick={done}>关闭</button><button className="button primary" disabled={sending}>{sending ? '发送中…' : '发送消息'}</button></div></form>;
}
