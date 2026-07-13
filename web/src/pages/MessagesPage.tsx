import { FormEvent, useEffect, useRef, useState } from 'react';
import { Empty, ErrorNotice, PageHeader, Status } from '../components/Common';
import { Icon } from '../components/Icon';
import { JsonViewer, parseMessageValue } from '../components/JsonViewer';
import { api } from '../lib/api';
import { MessageRecord } from '../lib/types';
import { activeFilterCount, filterQueryParams, JSONFilterCondition, JSONOperator, KeyOperator, MessageFilters, parseMessageFilters, replaceMessageFilterParams, ValueOperator } from './message-filters';

type MessageQueryResult = { items: MessageRecord[]; scanned: number; matched: number; skippedInvalidJson: number; resultLimited: boolean; scanLimited: boolean };
type QueryMeta = Omit<MessageQueryResult, 'items'> & { matched: number; filters: number };

export function MessagesPage({ clusterId, fixedTopic, embedded = false }: { clusterId: string; fixedTopic?: string; embedded?: boolean }) {
  const [items, setItems] = useState<MessageRecord[]>([]);
  const [selected, setSelected] = useState<MessageRecord>();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [producer, setProducer] = useState(false);
  const [live, setLive] = useState(false);
  const [liveFilterCount, setLiveFilterCount] = useState(0);
  const [topic, setTopic] = useState(fixedTopic || '');
  const [partition, setPartition] = useState(-1);
  const [mode, setMode] = useState('latest');
  const [offset, setOffset] = useState(0);
  const [timestamp, setTimestamp] = useState('');
  const [limit, setLimit] = useState(100);
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [filters, setFilters] = useState<MessageFilters>(() => parseMessageFilters(new URLSearchParams(window.location.search)));
  const [queryMeta, setQueryMeta] = useState<QueryMeta>();
  const stream = useRef<EventSource | null>(null);
  const filterCount = activeFilterCount(filters);

  const stop = () => { stream.current?.close(); stream.current = null; setLive(false); setLiveFilterCount(0); };
  async function queryMessages(requestedTopic = topic) {
    stop(); setLoading(true);
    try {
      const query = new URLSearchParams({ topic: requestedTopic, partition: String(partition), mode, limit: String(limit), offset: String(offset), timestamp: String(timestamp ? new Date(timestamp).getTime() : 0), ...filterQueryParams(filters) });
      const result = await api.get<MessageQueryResult>(`/api/v1/clusters/${clusterId}/messages?${query}`);
      setItems(result.items); setSelected(result.items[0]); setError('');
      setQueryMeta({ scanned: result.scanned ?? result.items.length, skippedInvalidJson: result.skippedInvalidJson ?? 0, resultLimited: Boolean(result.resultLimited), scanLimited: Boolean(result.scanLimited), matched: result.matched ?? result.items.length, filters: filterCount });
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
  useEffect(() => {
    const next = replaceMessageFilterParams(new URLSearchParams(window.location.search), filters);
    const query = next.toString();
    const url = `${window.location.pathname}${query ? `?${query}` : ''}${window.location.hash}`;
    window.history.replaceState(window.history.state, '', url);
  }, [filters]);

  function follow() {
    if (live) { stop(); return; }
    if (!topic.trim()) { setError('请先选择 Topic'); return; }
    setItems([]); setSelected(undefined); setError(''); setQueryMeta(undefined);
    const query = new URLSearchParams({ topic, partition: String(partition), ...filterQueryParams(filters) });
    const source = new EventSource(`/api/v1/clusters/${clusterId}/messages/stream?${query}`);
    stream.current = source;
    source.onmessage = (event) => { const record = JSON.parse(event.data) as MessageRecord; setItems((current) => [record, ...current].slice(0, 500)); };
    source.onerror = () => { setError('实时连接已中断，可重新开始跟随'); stop(); };
    setLiveFilterCount(filterCount); setLive(true);
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
      <label>最多返回<input type="number" min="1" max="500" value={limit} onChange={(event) => setLimit(Number(event.target.value))} /></label>
      <div className="query-actions"><button type="button" className={`button filter-button ${filterCount ? 'active' : 'ghost'}`} onClick={() => setFiltersOpen(!filtersOpen)}><Icon name="settings" />过滤{filterCount > 0 && <span>{filterCount}</span>}</button><button className="button primary" disabled={loading}><Icon name="search" />{loading ? '查询中' : '查询'}</button><button type="button" className={`button ${live ? 'danger' : 'ghost'}`} onClick={follow}><Icon name={live ? 'stop' : 'play'} />{live ? '停止' : '实时跟随'}</button></div>
    </form>
    {filtersOpen && <MessageFilterPanel filters={filters} setFilters={setFilters} close={() => setFiltersOpen(false)} />}
    {live && <div className="live-strip"><span />正在监听新消息，仅保留最近 500 条{liveFilterCount > 0 ? ` · 已应用 ${liveFilterCount} 条过滤条件` : ''}</div>}
    {queryMeta && queryMeta.filters > 0 && <div className="filter-result" role="status"><div><Icon name="search" /><span>已扫描 <b>{queryMeta.scanned.toLocaleString()}</b> 条，匹配 <b>{queryMeta.matched.toLocaleString()}</b> 条{queryMeta.resultLimited && <>，返回最新 <b>{items.length.toLocaleString()}</b> 条</>}</span></div><div>{queryMeta.skippedInvalidJson > 0 && <span className="warn-text">跳过 {queryMeta.skippedInvalidJson.toLocaleString()} 条非 JSON 消息</span>}{queryMeta.scanLimited && <span>已达到扫描上限</span>}{queryMeta.resultLimited && <span>已达到返回上限</span>}</div></div>}
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

function MessageFilterPanel({ filters, setFilters, close }: { filters: MessageFilters; setFilters: (next: MessageFilters | ((current: MessageFilters) => MessageFilters)) => void; close: () => void }) {
  const updateJSON = (id: string, patch: Partial<JSONFilterCondition>) => setFilters((current) => ({ ...current, jsonFilters: current.jsonFilters.map((filter) => filter.id === id ? { ...filter, ...patch } : filter) }));
  const addJSON = () => setFilters((current) => current.jsonFilters.length >= 5 ? current : ({ ...current, jsonFilters: [...current.jsonFilters, { id: `${Date.now()}-${current.jsonFilters.length}`, path: '', operator: 'eq', value: '' }] }));
  const removeJSON = (id: string) => setFilters((current) => ({ ...current, jsonFilters: current.jsonFilters.filter((filter) => filter.id !== id) }));
  const clear = () => setFilters({ keyFilter: '', keyOperator: 'contains', valueFilter: '', valueOperator: 'contains', scanLimit: 5000, jsonFilters: [] });
  return <section className="message-filters" aria-label="消息高级过滤">
    <header><div><span className="section-code">SERVER-SIDE FILTER</span><h3>高级过滤</h3><p>服务端扫描 Kafka 消息后过滤；多个条件使用 AND，字符串匹配区分大小写。</p></div><div><button type="button" className="button ghost" onClick={clear}>清空</button><button type="button" className="icon-button" aria-label="收起过滤条件" onClick={close}><Icon name="close" /></button></div></header>
    <div className="text-filter-grid">
      <label><span>Key</span><div><select aria-label="Key 匹配方式" value={filters.keyOperator} onChange={(event) => setFilters({ ...filters, keyOperator: event.target.value as KeyOperator })}><option value="contains">包含</option><option value="exact">精确等于</option><option value="prefix">前缀匹配</option></select><input value={filters.keyFilter} onChange={(event) => setFilters({ ...filters, keyFilter: event.target.value })} placeholder="例如 order-10086" /></div></label>
      <label><span>Value 文本</span><div><select aria-label="Value 匹配方式" value={filters.valueOperator} onChange={(event) => setFilters({ ...filters, valueOperator: event.target.value as ValueOperator })}><option value="contains">包含</option><option value="exact">精确等于</option></select><input value={filters.valueFilter} onChange={(event) => setFilters({ ...filters, valueFilter: event.target.value })} placeholder="适合非 JSON 或全文关键词" /></div></label>
    </div>
    <div className="json-filter-heading"><div><h4>JSON 字段条件</h4><span>支持嵌套路径和数组下标，例如 <code>data.user.id</code>、<code>items.0.sku</code></span></div><button type="button" className="button ghost" disabled={filters.jsonFilters.length >= 5} onClick={addJSON}><Icon name="plus" />添加条件</button></div>
    {filters.jsonFilters.length === 0 ? <button type="button" className="empty-filter" onClick={addJSON}><Icon name="plus" />添加第一个 JSON 字段条件</button> : <div className="json-filter-list">{filters.jsonFilters.map((filter, index) => <div className="json-filter-row" key={filter.id}><span className="and-badge">{index === 0 ? 'IF' : 'AND'}</span><label><span>字段路径</span><input value={filter.path} onChange={(event) => updateJSON(filter.id, { path: event.target.value })} placeholder="data.user.id" /></label><label><span>运算符</span><select value={filter.operator} onChange={(event) => updateJSON(filter.id, { operator: event.target.value as JSONOperator })}><option value="eq">等于</option><option value="neq">不等于</option><option value="contains">包含</option><option value="exists">字段存在</option><option value="gt">大于</option><option value="gte">大于等于</option><option value="lt">小于</option><option value="lte">小于等于</option></select></label><label><span>目标值</span><input value={filter.value} disabled={filter.operator === 'exists'} onChange={(event) => updateJSON(filter.id, { value: event.target.value })} placeholder={filter.operator === 'exists' ? '无需填写' : '10086 / SUCCESS / true'} /></label><button type="button" className="icon-button remove-filter" aria-label={`移除 JSON 条件 ${index + 1}`} onClick={() => removeJSON(filter.id)}><Icon name="trash" size={15} /></button></div>)}</div>}
    <footer><label>最多扫描<input type="number" min="1" max="50000" value={filters.scanLimit} onChange={(event) => setFilters({ ...filters, scanLimit: Math.min(50000, Math.max(1, Number(event.target.value))) })} /><span>条</span></label><p>扫描越多越可能找到历史消息，也会增加查询时间和 Kafka 读取量。最大 50,000 条。</p></footer>
  </section>;
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
