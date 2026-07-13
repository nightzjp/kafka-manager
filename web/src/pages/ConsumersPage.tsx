import { FormEvent, useEffect, useMemo, useState } from 'react';
import { ConfirmDialog, Drawer, Empty, ErrorNotice, Loading, PageHeader, Status } from '../components/Common';
import { useFeedback } from '../components/Feedback';
import { Icon } from '../components/Icon';
import { api } from '../lib/api';
import { ConsumerGroup, PartitionLag } from '../lib/types';
import { ConsumerSort, filterPartitions, sortConsumerGroups, summarizeConsumerGroups } from './consumers-model';

type HealthFilter = 'all' | 'lagging' | 'healthy';

export function ConsumersPage({ clusterId }: { clusterId: string }) {
  const [items, setItems] = useState<ConsumerGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<ConsumerGroup>();
  const [search, setSearch] = useState('');
  const [health, setHealth] = useState<HealthFilter>('all');
  const [sort, setSort] = useState<ConsumerSort>('lag-desc');

  const load = () => {
    if (!clusterId) return;
    setLoading(true);
    api.get<{ items: ConsumerGroup[] }>(`/api/v1/clusters/${clusterId}/consumer-groups`)
      .then((response) => {
        setItems(response.items.map((group) => ({ ...group, Partitions: group.Partitions || [] })));
        setError('');
      })
      .catch((reason: Error) => setError(reason.message))
      .finally(() => setLoading(false));
  };
  useEffect(load, [clusterId]);

  const summary = useMemo(() => summarizeConsumerGroups(items), [items]);
  const shown = useMemo(() => sortConsumerGroups(items.filter((group) => {
    const matchesName = group.Name.toLowerCase().includes(search.trim().toLowerCase());
    const matchesHealth = health === 'all' || (health === 'lagging' ? group.TotalLag > 0 : group.TotalLag === 0);
    return matchesName && matchesHealth;
  }), sort), [health, items, search, sort]);
  const maxLag = Math.max(1, ...shown.map((group) => group.TotalLag));

  return <>
    <PageHeader code="04 / CONSUMERS" title="消费组诊断" description="从积压最严重的消费组开始，逐层检查分区与 Offset 进度。" actions={<button className="button ghost" onClick={load}><Icon name="refresh" />重新计算 Lag</button>} />
    <section className="consumer-summary" aria-label="消费组汇总">
      <DiagnosticMetric label="消费组" value={summary.groups} detail="当前集群" />
      <DiagnosticMetric label="存在积压" value={summary.laggingGroups} detail="需要关注" tone={summary.laggingGroups ? 'warn' : 'good'} />
      <DiagnosticMetric label="Total Lag" value={summary.totalLag} detail="未消费消息" tone={summary.totalLag ? 'warn' : 'good'} />
      <DiagnosticMetric label="积压分区" value={summary.laggingPartitions} detail="受影响分区" tone={summary.laggingPartitions ? 'bad' : 'good'} />
    </section>
    <div className="consumer-toolbar">
      <label className="search-field"><Icon name="search" /><input aria-label="搜索消费组" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索消费组名称" /></label>
      <label><span>积压状态</span><select value={health} onChange={(event) => setHealth(event.target.value as HealthFilter)}><option value="all">全部</option><option value="lagging">仅有积压</option><option value="healthy">仅无积压</option></select></label>
      <label><span>排序</span><select value={sort} onChange={(event) => setSort(event.target.value as ConsumerSort)}><option value="lag-desc">Lag 从高到低</option><option value="members-desc">成员数从高到低</option><option value="name-asc">名称 A–Z</option></select></label>
      <strong>{shown.length} / {items.length}</strong>
    </div>
    {error && <ErrorNotice message={error} />}
    {loading ? <Loading /> : shown.length === 0 ? <Empty title="没有匹配的消费组" detail="调整搜索或积压筛选条件。" /> : <div className="consumer-table-wrap">
      <table className="consumer-table">
        <thead><tr><th>消费组</th><th>状态</th><th>成员</th><th>分区</th><th>总 Lag</th><th>积压强度</th><th><span className="sr-only">操作</span></th></tr></thead>
        <tbody>{shown.map((group) => {
          const lagging = group.Partitions.filter((partition) => partition.lag > 0).length;
          const tone = groupTone(group);
          return <tr key={group.Name}>
            <td><button className="consumer-name" onClick={() => setSelected(group)}><span className="consumer-avatar">CG</span><span><b>{group.Name}</b><small>{group.Protocol || '未声明协议'}</small></span></button></td>
            <td><Status tone={tone}>{group.State || 'Unknown'}</Status></td>
            <td><strong className="table-number">{group.MemberCount}</strong><small className="cell-caption">members</small></td>
            <td><strong className="table-number">{group.Partitions.length}</strong><small className={lagging ? 'cell-caption warn-text' : 'cell-caption'}>{lagging ? `${lagging} 个积压` : '全部正常'}</small></td>
            <td><strong className={group.TotalLag ? 'lag-number' : 'table-number'}>{group.TotalLag.toLocaleString()}</strong></td>
            <td><div className="consumer-lag"><div><i style={{ width: `${Math.max(group.TotalLag ? 4 : 0, group.TotalLag / maxLag * 100)}%` }} /></div><span>{lagLabel(group.TotalLag)}</span></div></td>
            <td><button className="consumer-open" onClick={() => setSelected(group)}>诊断 <Icon name="arrow" size={15} /></button></td>
          </tr>;
        })}</tbody>
      </table>
    </div>}
    {selected && <GroupDrawer group={selected} clusterId={clusterId} close={() => setSelected(undefined)} saved={() => { setSelected(undefined); load(); }} />}
  </>;
}

function DiagnosticMetric({ label, value, detail, tone = 'neutral' }: { label: string; value: number; detail: string; tone?: 'neutral' | 'good' | 'warn' | 'bad' }) {
  return <article className={`diagnostic-metric ${tone}`}><div><span>{label}</span><small>{detail}</small></div><strong>{value.toLocaleString()}</strong></article>;
}

function GroupDrawer({ group, clusterId, close, saved }: { group: ConsumerGroup; clusterId: string; close: () => void; saved: () => void }) {
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [lagOnly, setLagOnly] = useState(false);
  const [mode, setMode] = useState('latest');
  const [resetting, setResetting] = useState(false);
  const [resetRequest, setResetRequest] = useState<{ mode: string; offset: number }>();
  const feedback = useFeedback();
  const partitions = useMemo(() => filterPartitions(group.Partitions || [], search, lagOnly), [group.Partitions, lagOnly, search]);
  const laggingCount = group.Partitions.filter((partition) => partition.lag > 0).length;

  async function reset(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setError('');
    setResetRequest({ mode, offset: Number(data.get('offset') || 0) });
  }

  async function confirmReset() {
    if (!resetRequest) return;
    setResetting(true); setError('');
    try {
      await api.post(`/api/v1/clusters/${clusterId}/consumer-groups/${encodeURIComponent(group.Name)}/reset`, resetRequest);
      feedback.success('Consumer Offset 已重置', `${group.Name} · ${resetModeLabel(resetRequest.mode, resetRequest.offset)}`);
      setResetRequest(undefined);
      saved();
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : '重置失败';
      setError(message); feedback.error('重置 Offset 失败', message);
    }
    finally { setResetting(false); }
  }

  return <><Drawer title={group.Name} subtitle={`${clusterId} · ${group.Protocol || '未声明协议'}`} onClose={close}>
    <div className="consumer-drawer-body">
      <section className="drawer-metrics">
        <DrawerMetric label="状态" value={group.State || 'Unknown'} tone={groupTone(group)} />
        <DrawerMetric label="成员" value={group.MemberCount.toLocaleString()} />
        <DrawerMetric label="Total Lag" value={group.TotalLag.toLocaleString()} tone={group.TotalLag ? 'warn' : 'good'} />
        <DrawerMetric label="积压分区" value={`${laggingCount} / ${group.Partitions.length}`} tone={laggingCount ? 'bad' : 'good'} />
      </section>
      <section className="drawer-section">
        <div className="drawer-section-heading"><div><h3>分区进度</h3><p>按 Lag 从高到低排列，优先显示问题分区。</p></div><span>{partitions.length} 个分区</span></div>
        <div className="partition-toolbar"><label className="search-field compact"><Icon name="search" /><input aria-label="搜索 Topic 或 Partition" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Topic 或 P0" /></label><label className="check-field"><input type="checkbox" checked={lagOnly} onChange={(event) => setLagOnly(event.target.checked)} />仅看积压</label></div>
        {partitions.length === 0 ? <Empty title="没有匹配的分区" detail="调整搜索条件或关闭积压筛选。" /> : <div className="drawer-partitions"><table><thead><tr><th>Topic / Partition</th><th>Offset</th><th>进度</th><th>Lag</th></tr></thead><tbody>{partitions.map((partition) => <PartitionRow key={`${partition.topic}-${partition.partition}`} partition={partition} />)}</tbody></table></div>}
      </section>
      <section className="danger-zone">
        <div><span className="section-code">DANGER ZONE</span><h3>重置 Consumer Offset</h3><p>该操作会改变消费位置，可能导致消息重复消费或跳过。执行前请确认消费者已经停止。</p></div>
        <form className={mode === 'absolute' ? '' : 'compact'} onSubmit={reset}><label>目标位置<select value={mode} onChange={(event) => setMode(event.target.value)}><option value="earliest">重置到最早</option><option value="latest">重置到最新</option><option value="absolute">指定 Offset</option></select></label>{mode === 'absolute' && <label>Offset<input name="offset" type="number" min="0" required placeholder="输入绝对 Offset" /></label>}<button className="button danger" disabled={resetting}><Icon name="warning" />{resetting ? '正在重置…' : '重置 Offset'}</button></form>
        {error && <ErrorNotice message={error} />}
      </section>
    </div>
  </Drawer>{resetRequest && <ConfirmDialog title="重置 Consumer Offset" description={<><b>消费位置将被修改为“{resetModeLabel(resetRequest.mode, resetRequest.offset)}”。</b><p>可能导致消息重复消费或被跳过。请先确认该消费组的消费者已经停止。</p></>} confirmLabel="确认重置 Offset" confirmationText={group.Name} pending={resetting} error={error} onClose={() => { setResetRequest(undefined); setError(''); }} onConfirm={confirmReset} />}</>;
}

function DrawerMetric({ label, value, tone = 'neutral' }: { label: string; value: string; tone?: 'neutral' | 'good' | 'warn' | 'bad' }) {
  return <article className={`drawer-metric ${tone}`}><span>{label}</span><strong>{value}</strong></article>;
}

function PartitionRow({ partition }: { partition: PartitionLag }) {
  const progress = partition.endOffset <= 0 ? (partition.lag ? 0 : 100) : Math.max(0, Math.min(100, partition.currentOffset / partition.endOffset * 100));
  return <tr><td><b>{partition.topic}</b><code>P{partition.partition}</code></td><td><code>{partition.currentOffset.toLocaleString()} → {partition.endOffset.toLocaleString()}</code></td><td><div className="offset-progress"><i style={{ width: `${progress}%` }} /></div><small>{progress.toFixed(1)}%</small></td><td><strong className={partition.lag ? 'lag-number' : ''}>{partition.lag.toLocaleString()}</strong></td></tr>;
}

function groupTone(group: ConsumerGroup): 'good' | 'warn' | 'bad' | 'neutral' {
  if (/dead|unknown/i.test(group.State)) return 'bad';
  if (group.TotalLag > 0 || /preparing|completing/i.test(group.State)) return 'warn';
  if (/stable/i.test(group.State)) return 'good';
  return 'neutral';
}

function lagLabel(lag: number) { if (!lag) return '无积压'; if (lag >= 10000) return '严重'; if (lag >= 1000) return '较高'; return '待关注'; }

function resetModeLabel(mode: string, offset: number) {
  if (mode === 'earliest') return '最早位置';
  if (mode === 'absolute') return `Offset ${offset}`;
  return '最新位置';
}
