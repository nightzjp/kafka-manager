import { FormEvent, useEffect, useState } from 'react';
import { ConfirmDialog, ErrorNotice, Status, Tabs } from '../components/Common';
import { useFeedback } from '../components/Feedback';
import { Icon } from '../components/Icon';
import { api } from '../lib/api';
import { Topic, TopicConfig } from '../lib/types';
import { MessagesPage } from './MessagesPage';

export type TopicTab = 'overview' | 'messages' | 'partitions' | 'config';

export function TopicWorkspace({ clusterId, topic, tab, setTab, onBack }: {
  clusterId: string;
  topic: Topic;
  tab: TopicTab;
  setTab: (tab: TopicTab) => void;
  onBack: () => void;
}) {
  const [error, setError] = useState('');
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
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
      <button className="button danger" onClick={() => { setError(''); setConfirmingDelete(true); }}><Icon name="trash" />删除 Topic</button>
    </header>
    <Tabs value={tab} onChange={setTab} items={[
      { id: 'overview', label: '概览' }, { id: 'messages', label: '消息' },
      { id: 'partitions', label: '分区', count: topic.PartitionCount }, { id: 'config', label: '配置' },
    ]} />
    {error && <ErrorNotice message={error} />}
    <div className="tab-panel" role="tabpanel" id={`workspace-panel-${tab}`} aria-labelledby={`workspace-tab-${tab}`}>
      {tab === 'overview' && <Overview topic={topic} setTab={setTab} />}
      {tab === 'messages' && <MessagesPage clusterId={clusterId} fixedTopic={topic.Name} embedded />}
      {tab === 'partitions' && <Partitions clusterId={clusterId} topic={topic} />}
      {tab === 'config' && <Configs clusterId={clusterId} topic={topic.Name} />}
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

function Partitions({ clusterId, topic }: { clusterId: string; topic: Topic }) {
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [adding, setAdding] = useState(false);
  const feedback = useFeedback();
  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const count = Number(new FormData(event.currentTarget).get('count'));
    try {
      await api.post(`/api/v1/clusters/${clusterId}/topics/${encodeURIComponent(topic.Name)}/partitions`, { count });
      setAdding(false); setError('');
      setNotice(`已提交新增 ${count} 个分区；返回 Topic 列表后会刷新最新元数据。`);
      feedback.success('分区扩容已提交', `${topic.Name} 新增 ${count} 个分区`);
    } catch (reason) { setError(reason instanceof Error ? reason.message : '扩容失败'); }
  }
  return <section className="panel">
    <div className="panel-heading"><div><h2>分区列表</h2><p>扩分区不可撤销，请确认生产者分区策略</p></div><button className="button primary" onClick={() => setAdding(!adding)}><Icon name="plus" />增加分区</button></div>
    {adding && <form className="inline-form" onSubmit={add}><label>新增数量<input name="count" type="number" min="1" defaultValue="1" /></label><button className="button primary">确认扩容</button></form>}
    {notice && <div className="success-notice" role="status"><Icon name="check" />{notice}</div>}
    {error && <ErrorNotice message={error} />}
    <div className="data-table"><table><thead><tr><th>Partition</th><th>Leader</th><th>Replicas</th><th>ISR</th><th>Offline</th></tr></thead><tbody>
      {topic.Partitions.map((partition) => <tr key={partition.ID}><td className="mono">P{partition.ID}</td><td className="mono">{partition.Leader}</td><td className="mono">{partition.Replicas.join(', ')}</td><td className="mono">{partition.ISR.join(', ')}</td><td>{partition.OfflineReplicas.length ? <Status tone="bad">{partition.OfflineReplicas.join(', ')}</Status> : <Status tone="good">无</Status>}</td></tr>)}
    </tbody></table></div>
  </section>;
}

function Configs({ clusterId, topic }: { clusterId: string; topic: string }) {
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
    <form className="config-form" onSubmit={alter}><label>配置项<input name="name" required placeholder="retention.ms" /></label><label>配置值<input name="value" placeholder="留空恢复默认" /></label><button className="button primary">应用配置</button></form>
    {error && <ErrorNotice message={error} />}
    <div className="config-list">{shown.map((item) => <div key={item.name}><code>{item.name}</code><span>{item.sensitive ? '••••••' : item.value ?? '默认'}</span><small>{item.source}</small></div>)}</div>
  </section>;
}
