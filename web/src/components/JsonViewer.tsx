import { ReactElement, useMemo, useState } from 'react';
import { Icon } from './Icon';

export type ParsedValue = { kind: 'json'; value: unknown } | { kind: 'text'; value: string };
export function parseMessageValue(raw: string): ParsedValue { try { return { kind: 'json', value: JSON.parse(raw) }; } catch { return { kind: 'text', value: raw }; } }
export function formatJson(value: unknown) { return JSON.stringify(value, null, 2); }

export function JsonViewer({ raw }: { raw: string }) {
  const parsed = useMemo(() => parseMessageValue(raw), [raw]);
  const [view, setView] = useState<'pretty' | 'raw'>('pretty');
  const [collapsed, setCollapsed] = useState(false);
  const copy = () => navigator.clipboard?.writeText(parsed.kind === 'json' ? formatJson(parsed.value) : raw);
  return <section className="json-viewer"><header><div><span className={`type-badge ${parsed.kind}`}>{parsed.kind.toUpperCase()}</span><span>{parsed.kind === 'json' ? '已格式化' : '原始文本'}</span></div><div>{parsed.kind === 'json' && <><button type="button" onClick={() => setCollapsed(!collapsed)}>{collapsed ? '展开' : '折叠'}</button><button type="button" onClick={() => setView(view === 'pretty' ? 'raw' : 'pretty')}>{view === 'pretty' ? '原始' : '格式化'}</button></>}<button type="button" onClick={copy}><Icon name="copy" size={14} />复制</button></div></header><div className="json-content">{parsed.kind === 'json' && view === 'pretty' ? collapsed ? <code>{summary(parsed.value)}</code> : <JsonNode value={parsed.value} /> : <pre>{raw}</pre>}</div></section>;
}
function summary(value: unknown) { if (Array.isArray(value)) return `Array(${value.length})`; if (value && typeof value === 'object') return `Object(${Object.keys(value).length})`; return String(value); }
function JsonNode({ value, depth = 0 }: { value: unknown; depth?: number }): ReactElement {
  const [collapsed, setCollapsed] = useState(false);
  if (value === null) return <span className="json-null">null</span>;
  if (typeof value === 'string') return <span className="json-string">{JSON.stringify(value)}</span>;
  if (typeof value === 'number') return <span className="json-number">{value}</span>;
  if (typeof value === 'boolean') return <span className="json-boolean">{String(value)}</span>;
  const entries = Array.isArray(value) ? value.map((item, index) => [String(index), item] as const) : Object.entries(value as Record<string, unknown>);
  const open = Array.isArray(value) ? '[' : '{', close = Array.isArray(value) ? ']' : '}';
  return <span className="json-node"><button type="button" className="json-toggle" aria-label={collapsed ? '展开节点' : '折叠节点'} onClick={() => setCollapsed(!collapsed)}>{collapsed ? '▸' : '▾'}</button>{open}{collapsed ? <span className="json-muted"> {entries.length} 项 </span> : <div>{entries.map(([key, item], index) => <div className="json-line" style={{ paddingLeft: `${Math.min(depth + 1, 8) * 16}px` }} key={key}><span className="json-key">{Array.isArray(value) ? key : JSON.stringify(key)}</span><span>: </span><JsonNode value={item} depth={depth + 1} />{index < entries.length - 1 && ','}</div>)}</div>}{close}</span>;
}
