import { FormEvent, useDeferredValue, useEffect, useState } from 'react';
import { api } from '../lib/api';
import { Topic } from '../lib/types';
import { Dialog, Empty, ErrorNotice, Loading, PageHeader, Status } from '../components/Common';
import { Icon } from '../components/Icon';
import { useFeedback } from '../components/Feedback';

export function TopicsPage({ clusterId, onOpen }: { clusterId: string; onOpen: (topic: Topic, tab?: 'overview' | 'messages') => void }) {
  const [items, setItems] = useState<Topic[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const deferred = useDeferredValue(search);
  const feedback = useFeedback();
  const load = () => {
    if (!clusterId) return;
    setLoading(true);
    api.get<{ items: Topic[] }>(`/api/v1/clusters/${clusterId}/topics?search=${encodeURIComponent(deferred)}&pageSize=200`)
      .then((response) => { setItems(response.items); setError(''); })
      .catch((reason: Error) => setError(reason.message))
      .finally(() => setLoading(false));
  };
  useEffect(load, [clusterId, deferred]);

  async function copyName(name: string) {
    try {
      if (!navigator.clipboard) throw new Error('当前浏览器不支持剪贴板 API');
      await navigator.clipboard.writeText(name);
      feedback.success('Topic 名称已复制', name);
    } catch (reason) {
      feedback.error('复制失败', reason instanceof Error ? reason.message : '请手动选择 Topic 名称');
    }
  }

  return <>
    <PageHeader code="TOPICS" title="Topic 工作台" description="查看健康状态，进入 Topic 后直接读取消息、检查分区和修改配置。" actions={<button className="button primary" onClick={() => setCreating(true)}><Icon name="plus" />创建 Topic</button>} />
    <div className="toolbar"><label className="search-field"><Icon name="search" /><input aria-label="搜索 Topic" placeholder="按名称筛选 Topic" value={search} onChange={(event) => setSearch(event.target.value)} /></label><span>{items.length} 个 Topic</span></div>
    {error && <ErrorNotice message={error} />}
    {loading ? <Loading /> : items.length === 0 ? <Empty title="没有匹配的 Topic" detail="调整搜索条件，或创建第一个 Topic。" /> : <div className="data-table"><table><thead><tr><th>Topic</th><th>健康状态</th><th>分区</th><th>复制因子</th><th>操作</th></tr></thead><tbody>{items.map((topic) => <tr key={topic.Name}>
      <td><div className="topic-name"><span className="topic-glyph">T</span><div><button className="topic-link" onClick={() => onOpen(topic)}>{topic.Name}</button><small>{topic.Internal ? '系统 Topic' : '业务 Topic'}</small></div></div></td>
      <td>{topic.UnderReplicated ? <Status tone="bad">{topic.UnderReplicated} 个 ISR 异常</Status> : <Status tone="good">健康</Status>}</td>
      <td className="mono">{topic.PartitionCount}</td><td className="mono">{topic.ReplicationFactor}</td>
      <td><div className="row-actions"><button aria-label={`复制 ${topic.Name}`} onClick={() => void copyName(topic.Name)} title="复制名称"><Icon name="copy" /></button><button onClick={() => onOpen(topic, 'messages')}>查看消息 <Icon name="arrow" /></button></div></td>
    </tr>)}</tbody></table></div>}
    {creating && <CreateTopic clusterId={clusterId} close={() => setCreating(false)} saved={(name) => { setCreating(false); feedback.success('Topic 创建成功', name); load(); }} />}
  </>;
}

function CreateTopic({ clusterId, close, saved }: { clusterId: string; close: () => void; saved: (name: string) => void }) {
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const name = String(data.get('name') || '');
    setSubmitting(true); setError('');
    try {
      await api.post(`/api/v1/clusters/${clusterId}/topics`, { name, partitions: Number(data.get('partitions')), replicationFactor: Number(data.get('replication')) });
      saved(name);
    } catch (reason) { setError(reason instanceof Error ? reason.message : '创建失败'); }
    finally { setSubmitting(false); }
  }
  return <Dialog title="创建 Topic" onClose={close}><form className="dialog-body form" onSubmit={submit}><label>Topic 名称<input name="name" required placeholder="orders.v1" autoFocus /></label><div className="form-row"><label>分区数<input name="partitions" type="number" min="1" defaultValue="3" /></label><label>复制因子<input name="replication" type="number" min="1" defaultValue="1" /></label></div>{error && <ErrorNotice message={error} />}<div className="form-actions"><button type="button" className="button ghost" disabled={submitting} onClick={close}>取消</button><button className="button primary" disabled={submitting}>{submitting ? '正在创建…' : '创建 Topic'}</button></div></form></Dialog>;
}
