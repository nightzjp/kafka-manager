import { FormEvent, useEffect, useState } from 'react';
import { ConfirmDialog, ErrorNotice, Status, Tabs } from '../components/Common';
import { useFeedback } from '../components/Feedback';
import { Icon } from '../components/Icon';
import { api } from '../lib/api';
import { Topic, TopicConfig } from '../lib/types';
import { MessagesPage } from './MessagesPage';

export type TopicTab = 'overview' | 'messages' | 'partitions' | 'config';

export function TopicWorkspace({ clusterId, topic, tab, setTab, onBack, onRefresh, readOnly }: {
  clusterId: string;
  topic: Topic;
  tab: TopicTab;
  setTab: (tab: TopicTab) => void;
  onBack: () => void;
  onRefresh: () => Promise<void>;
  readOnly: boolean;
}) {
  const [error, setError] = useState('');
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const feedback = useFeedback();
  async function remove() {
    setDeleting(true); setError('');
    try {
      await api.delete(`/api/v1/clusters/${clusterId}/topics/${encodeURIComponent(topic.Name)}`);
      feedback.success('Topic 已删除', topic.Name);
      setConfirmingDelete(false);
      onBack();
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : '删除 Topic 失败';
      setError(message); feedback.error('删除 Topic 失败', message);
    } finally { setDeleting(false); }
  }
  async function refresh() {
    setRefreshing(true); setError('');
    try { await onRefresh(); feedback.success('Topic 已刷新', '分区、ISR 和复制因子已更新'); }
    catch (reason) { const message = reason instanceof Error ? reason.message : '刷新 Topic 失败'; setError(message); feedback.error('刷新 Topic 失败', message); }
    finally { setRefreshing(false); }
  }

  return <div className="topic-workspace">
    <button className="back-link" onClick={onBack}><Icon name="back" />返回 Topics</button>
    <header className="workspace-heading">
      <div>
        <div className="breadcrumb">TOPICS / {clusterId}</div>
        <h1>{topic.Name}</h1>
        <div className="heading-meta">
          {topic.UnderReplicated ? <Status tone="bad">ISR 异常</Status> : <Status tone="good">运行正常</Status>}
          <span>{topic.PartitionCount} 分区</span><span>复制因子 {topic.ReplicationFactor}</span>
        </div>
      </div>
      <div className="workspace-actions"><button className="button ghost" disabled={refreshing} onClick={() => void refresh()}><Icon name="refresh" />{refreshing ? '刷新中…' : '刷新'}</button><button className="button danger" disabled={readOnly} title={readOnly ? '当前集群为只读模式' : undefined} onClick={() => { setError(''); setConfirmingDelete(true); }}><Icon name="trash" />{readOnly ? '只读集群' : '删除 Topic'}</button></div>
    </header>
    <Tabs value={tab} onChange={setTab} items={[
      { id: 'overview', label: '概览' }, { id: 'messages', label: '消息' },
      { id: 'partitions', label: '分区', count: topic.PartitionCount }, { id: 'config', label: '配置' },
    ]} />
    {readOnly && <div className="readonly-notice"><Icon name="warning" /><span><b>当前集群为只读模式</b>，可以查看 Topic、消息、分区和配置，但不能执行 Kafka 写操作。</span></div>}
    {error && <ErrorNotice message={error} />}
    <div className="tab-panel" role="tabpanel" id={`workspace-panel-${tab}`} aria-labelledby={`workspace-tab-${tab}`}>
      {tab === 'overview' && <Overview topic={topic} setTab={setTab} />}
      {tab === 'messages' && <MessagesPage clusterId={clusterId} fixedTopic={topic.Name} embedded readOnly={readOnly} />}
      {tab === 'partitions' && <Partitions clusterId={clusterId} topic={topic} onRefresh={onRefresh} readOnly={readOnly} />}
      {tab === 'config' && <Configs clusterId={clusterId} topic={topic.Name} readOnly={readOnly} />}
    </div>
    {confirmingDelete && <ConfirmDialog title="删除 Topic" description={<><b>此操作不可撤销。</b><p>Topic 的全部消息、分区和配置都会被永久删除。</p></>} confirmLabel="永久删除 Topic" confirmationText={topic.Name} pending={deleting} error={error} onClose={() => { setConfirmingDelete(false); setError(''); }} onConfirm={remove} />}
  </div>;
}

function Overview({ topic, setTab }: { topic: Topic; setTab: (tab: TopicTab) => void }) {
  return <>
    <div className="summary-grid">
      <Summary label="分区" value={topic.PartitionCount} /><Summary label="复制因子" value={topic.ReplicationFactor} />
      <Summary label="ISR 异常" value={topic.UnderReplicated} tone={topic.UnderReplicated ? 'bad' : 'good'} />
      <Summary label="类型" value={topic.Internal ? '系统' : '业务'} />
    </div>
    <section className="panel">
      <div className="panel-heading"><div><h2>分区健康</h2><p>Leader 与 ISR 的实时分布</p></div><button className="button ghost" onClick={() => setTab('partitions')}>查看全部</button></div>
      <div className="partition-cards">{topic.Partitions.slice(0, 12).map((partition) => <article key={partition.ID}><b>P{partition.ID}</b><span>Leader {partition.Leader}</span><small>ISR {partition.ISR.join(', ')}</small></article>)}</div>
    </section>
  </>;
}

function Summary({ label, value, tone }: { label: string; value: string | number; tone?: string }) {
  return <article className={`summary-card ${tone || ''}`}><span>{label}</span><strong>{value}</strong></article>;
}

function Partitions({ clusterId, topic, onRefresh, readOnly }: { clusterId: string; topic: Topic; onRefresh: () => Promise<void>; readOnly: boolean }) {
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [adding, setAdding] = useState(false);
  const feedback = useFeedback();
  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const count = Number(new FormData(event.currentTarget).get('count'));
    let expanded = false;
    try {
      await api.post(`/api/v1/clusters/${clusterId}/topics/${encodeURIComponent(topic.Name)}/partitions`, { count });
      expanded = true;
      await onRefresh();
      setAdding(false); setError('');
      setNotice(`已新增 ${count} 个分区，当前分区列表和 Topic 元数据已刷新。`);
      feedback.success('分区扩容完成', `${topic.Name} 新增 ${count} 个分区`);
    } catch (reason) {
      const detail = reason instanceof Error ? reason.message : '未知错误';
      setError(expanded ? `分区扩容已提交，但刷新元数据失败：${detail}` : detail);
    }
  }
  return <section className="panel">
    <div className="panel-heading"><div><h2>分区列表</h2><p>扩分区不可撤销，请确认生产者分区策略</p></div><button className="button primary" disabled={readOnly} title={readOnly ? '当前集群为只读模式' : undefined} onClick={() => setAdding(!adding)}><Icon name="plus" />{readOnly ? '只读集群' : '增加分区'}</button></div>
    {adding && <form className="inline-form" onSubmit={add}><label>新增数量<input name="count" type="number" min="1" defaultValue="1" /></label><button className="button primary">确认扩容</button></form>}
    {notice && <div className="success-notice" role="status"><Icon name="check" />{notice}</div>}
    {error && <ErrorNotice message={error} />}
    <div className="data-table"><table><thead><tr><th>Partition</th><th>Leader</th><th>Replicas</th><th>ISR</th><th>Offline</th></tr></thead><tbody>
      {topic.Partitions.map((partition) => <tr key={partition.ID}><td className="mono">P{partition.ID}</td><td className="mono">{partition.Leader}</td><td className="mono">{partition.Replicas.join(', ')}</td><td className="mono">{partition.ISR.join(', ')}</td><td>{partition.OfflineReplicas.length ? <Status tone="bad">{partition.OfflineReplicas.join(', ')}</Status> : <Status tone="good">无</Status>}</td></tr>)}
    </tbody></table></div>
  </section>;
}

function Configs({ clusterId, topic, readOnly }: { clusterId: string; topic: string; readOnly: boolean }) {
  const [items, setItems] = useState<TopicConfig[]>([]);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const feedback = useFeedback();
  const load = () => { void api.get<{ items: TopicConfig[] }>(`/api/v1/clusters/${clusterId}/topics/${encodeURIComponent(topic)}/configs`).then((result) => { setItems(result.items); setError(''); }).catch((reason: Error) => setError(reason.message)); };
  useEffect(() => { load(); }, [clusterId, topic]);
  async function alter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget); const name = String(data.get('name') || '').trim(); const raw = String(data.get('value') || '');
    if (!name) return;
    try { await api.put(`/api/v1/clusters/${clusterId}/topics/${encodeURIComponent(topic)}/configs`, { configs: { [name]: raw === '' ? null : raw } }); event.currentTarget.reset(); feedback.success('Topic 配置已更新', name); load(); }
    catch (reason) { setError(reason instanceof Error ? reason.message : '配置失败'); }
  }
  const shown = items.filter((item) => item.name.toLowerCase().includes(search.toLowerCase()));
  return <section className="panel">
    <div className="panel-heading"><div><h2>Topic 配置</h2><p>留空配置值会恢复 Broker 默认值</p></div><label className="search-field compact"><Icon name="search" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="筛选配置项" /></label></div>
    <form className="config-form" onSubmit={alter}><label>配置项<input name="name" required disabled={readOnly} placeholder="retention.ms" /></label><label>配置值<input name="value" disabled={readOnly} placeholder="留空恢复默认" /></label><button className="button primary" disabled={readOnly} title={readOnly ? '当前集群为只读模式' : undefined}>{readOnly ? '只读集群' : '应用配置'}</button></form>
    {error && <ErrorNotice message={error} />}
    <div className="config-list">{shown.map((item) => <div key={item.name}><code>{item.name}</code><span>{item.sensitive ? '••••••' : item.value ?? '默认'}</span><small>{item.source}</small></div>)}</div>
  </section>;
}
