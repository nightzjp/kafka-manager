import { FormEvent, KeyboardEvent, ReactNode, useEffect, useRef, useState } from 'react';
import { Icon } from './Icon';
import { confirmationMatches } from './confirmation-model';

export function PageHeader({ code, title, description, actions }: { code: string; title: string; description: string; actions?: ReactNode }) {
  return <header className="page-header"><div><span className="section-code">{code}</span><h1>{title}</h1><p>{description}</p></div>{actions && <div className="page-actions">{actions}</div>}</header>;
}
export function Empty({ title, detail }: { title: string; detail: string }) { return <div className="empty"><div className="empty-mark">—</div><h3>{title}</h3><p>{detail}</p></div>; }
export function ErrorNotice({ message }: { message: string }) { return <div className="error-notice" role="alert"><Icon name="warning" /><div><b>操作未完成</b><span>{message}</span></div></div>; }
export function Loading() { return <div className="loading"><i /><i /><i /><span>正在读取 Kafka 元数据</span></div>; }
export function Status({ tone, children }: { tone: 'good' | 'warn' | 'bad' | 'neutral'; children: ReactNode }) { return <span className={`status ${tone}`}><i />{children}</span>; }

export function Tabs<T extends string>({ items, value, onChange, id = 'workspace' }: {
  items: { id: T; label: string; count?: number }[];
  value: T;
  onChange: (value: T) => void;
  id?: string;
}) {
  const refs = useRef<(HTMLButtonElement | null)[]>([]);
  function keyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    const next = (index + (event.key === 'ArrowRight' ? 1 : -1) + items.length) % items.length;
    onChange(items[next].id);
    refs.current[next]?.focus();
  }
  return <div className="tabs" role="tablist">{items.map((item, index) => <button
    ref={(node) => { refs.current[index] = node; }}
    id={`${id}-tab-${item.id}`}
    aria-controls={`${id}-panel-${item.id}`}
    role="tab"
    tabIndex={value === item.id ? 0 : -1}
    aria-selected={value === item.id}
    className={value === item.id ? 'active' : ''}
    key={item.id}
    onKeyDown={(event) => keyDown(event, index)}
    onClick={() => onChange(item.id)}
  >{item.label}{item.count !== undefined && <span>{item.count}</span>}</button>)}</div>;
}

export function Dialog({ title, children, onClose, closeDisabled = false }: { title: string; children: ReactNode; onClose: () => void; closeDisabled?: boolean }) {
  const dialog = useRef<HTMLElement>(null);
  useEffect(() => {
    const oldOverflow = document.body.style.overflow;
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    document.body.style.overflow = 'hidden';
    const focusable = () => Array.from(dialog.current?.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])') || []);
    focusable()[0]?.focus();
    const key = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') { if (!closeDisabled) onClose(); return; }
      if (event.key !== 'Tab') return;
      const items = focusable();
      if (!items.length) return;
      const first = items[0], last = items[items.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    addEventListener('keydown', key);
    return () => { document.body.style.overflow = oldOverflow; removeEventListener('keydown', key); previous?.focus(); };
  }, [closeDisabled, onClose]);
  return <div className="scrim" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && !closeDisabled && onClose()}><section ref={dialog} className="dialog" role="dialog" aria-modal="true" aria-label={title}><header><span>{title}</span><button className="icon-button" aria-label="关闭" disabled={closeDisabled} onClick={onClose}><Icon name="close" /></button></header>{children}</section></div>;
}

export function ConfirmDialog({ title, description, confirmLabel, confirmationText, pending = false, error, onConfirm, onClose }: {
  title: string;
  description: ReactNode;
  confirmLabel: string;
  confirmationText?: string;
  pending?: boolean;
  error?: string;
  onConfirm: () => void | Promise<void>;
  onClose: () => void;
}) {
  const [value, setValue] = useState('');
  const enabled = confirmationMatches(value, confirmationText) && !pending;
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (enabled) void onConfirm();
  }
  return <Dialog title={title} closeDisabled={pending} onClose={onClose}>
    <form className="dialog-body confirm-dialog" onSubmit={submit}>
      <div className="confirm-warning"><Icon name="warning" size={22} /><div>{description}</div></div>
      {confirmationText !== undefined && <label>输入 <code>{confirmationText}</code> 继续
        <input value={value} onChange={(event) => setValue(event.target.value)} autoComplete="off" autoFocus placeholder={confirmationText} />
      </label>}
      {error && <ErrorNotice message={error} />}
      <div className="form-actions"><button type="button" className="button ghost" disabled={pending} onClick={onClose}>取消</button><button className="button danger solid" disabled={!enabled}>{pending ? '正在执行…' : confirmLabel}</button></div>
    </form>
  </Dialog>;
}

export function Drawer({ title, subtitle, children, onClose }: { title: string; subtitle?: string; children: ReactNode; onClose: () => void }) {
  const drawer = useRef<HTMLElement>(null);
  useEffect(() => {
    const oldOverflow = document.body.style.overflow;
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    document.body.style.overflow = 'hidden';
    const focusable = () => Array.from(drawer.current?.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])') || []);
    focusable()[0]?.focus();
    const key = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') { onClose(); return; }
      if (event.key !== 'Tab') return;
      const items = focusable();
      if (!items.length) return;
      const first = items[0], last = items[items.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    addEventListener('keydown', key);
    return () => { document.body.style.overflow = oldOverflow; removeEventListener('keydown', key); previous?.focus(); };
  }, [onClose]);
  return <div className="drawer-scrim" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
    <aside ref={drawer} className="drawer" role="dialog" aria-modal="true" aria-label={title}>
      <header className="drawer-header"><div><span className="section-code">CONSUMER GROUP</span><h2>{title}</h2>{subtitle && <p>{subtitle}</p>}</div><button className="icon-button" aria-label="关闭" onClick={onClose}><Icon name="close" /></button></header>
      {children}
    </aside>
  </div>;
}
