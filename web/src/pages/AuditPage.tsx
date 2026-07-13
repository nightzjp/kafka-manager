import { useEffect, useMemo, useState } from 'react';
import { Empty, ErrorNotice, Loading, PageHeader } from '../components/Common';
import { Icon } from '../components/Icon';
import { api } from '../lib/api';
import { AuditEntry } from '../lib/types';

export function AuditPage() {
  const [items, setItems] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [result, setResult] = useState('all');
  const load = () => {
    setLoading(true);
    api.get<{ items: AuditEntry[] }>('/api/v1/audit?limit=200')
      .then((response) => { setItems(response.items); setError(''); })
      .catch((reason: Error) => setError(reason.message))
      .finally(() => setLoading(false));
  };
  useEffect(load, []);
  const shown = useMemo(() => items.filter((entry) => {
    const matchesResult = result === 'all' || entry.result === result;
    const haystack = `${entry.action} ${entry.clusterId} ${entry.resource} ${entry.error || ''}`.toLowerCase();
    return matchesResult && haystack.includes(search.trim().toLowerCase());
  }), [items, result, search]);

  return <>
    <PageHeader code="06 / AUDIT" title="操作时间线" description="按日期目录持久化管理操作，不保存消息正文和认证凭证。" actions={<button className="button ghost" onClick={load}><Icon name="refresh" />刷新日志</button>} />
    <div className="toolbar">
      <label className="search-field"><Icon name="search" /><input aria-label="搜索审计日志" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索操作、集群或资源" /></label>
      <select aria-label="筛选执行结果" value={result} onChange={(event) => setResult(event.target.value)}><option value="all">全部结果</option><option value="success">仅成功</option><option value="failure">仅失败</option></select>
      <span>{shown.length} 条记录</span>
    </div>
    {error && <ErrorNotice message={error} />}
    {loading ? <Loading /> : shown.length === 0
      ? <Empty title="没有匹配的操作记录" detail="调整筛选条件，或执行 Kafka 管理操作后再查看。" />
      : <div className="timeline">{shown.map((entry, index) => <article key={`${entry.timestamp}-${index}`}>
        <time>{new Date(entry.timestamp).toLocaleString()}</time>
        <span className={entry.result === 'success' ? 'success' : 'failure'} />
        <div><b>{entry.action}</b><code>{entry.clusterId || 'system'} / {entry.resource || '-'}</code>{entry.error && <p>{entry.error}</p>}</div>
        <strong>{entry.result === 'success' ? '成功' : '失败'}</strong>
      </article>)}</div>}
  </>;
}
