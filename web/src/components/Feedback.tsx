import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { Icon } from './Icon';
import { addFeedback, dismissFeedback, FeedbackItem, FeedbackTone } from './feedback-model';

type FeedbackAPI = {
  notify: (tone: FeedbackTone, title: string, message?: string, duration?: number) => void;
  success: (title: string, message?: string) => void;
  error: (title: string, message?: string) => void;
  info: (title: string, message?: string) => void;
};

const FeedbackContext = createContext<FeedbackAPI | null>(null);

export function FeedbackProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<FeedbackItem[]>([]);
  const serial = useRef(0);
  const notify = useCallback((tone: FeedbackTone, title: string, message = '', duration = 4500) => {
    const id = `${Date.now()}-${serial.current++}`;
    setItems((current) => addFeedback(current, { id, tone, title, message, duration }));
  }, []);
  const value = useMemo<FeedbackAPI>(() => ({
    notify,
    success: (title, message) => notify('success', title, message),
    error: (title, message) => notify('error', title, message, 6500),
    info: (title, message) => notify('info', title, message),
  }), [notify]);
  const dismiss = useCallback((id: string) => setItems((current) => dismissFeedback(current, id)), []);
  return <FeedbackContext.Provider value={value}>
    {children}
    <div className="feedback-stack" aria-label="操作通知">
      {items.map((item) => <FeedbackToast key={item.id} item={item} dismiss={dismiss} />)}
    </div>
  </FeedbackContext.Provider>;
}

function FeedbackToast({ item, dismiss }: { item: FeedbackItem; dismiss: (id: string) => void }) {
  useEffect(() => {
    const timer = window.setTimeout(() => dismiss(item.id), item.duration);
    return () => window.clearTimeout(timer);
  }, [dismiss, item.duration, item.id]);
  return <article className={`feedback-toast ${item.tone}`} role={item.tone === 'error' ? 'alert' : 'status'}>
    <span className="feedback-icon"><Icon name={item.tone === 'success' ? 'check' : item.tone === 'error' ? 'warning' : 'message'} /></span>
    <div><b>{item.title}</b>{item.message && <p>{item.message}</p>}</div>
    <button aria-label="关闭通知" onClick={() => dismiss(item.id)}><Icon name="close" size={15} /></button>
  </article>;
}

export function useFeedback(): FeedbackAPI {
  const value = useContext(FeedbackContext);
  if (!value) throw new Error('useFeedback 必须在 FeedbackProvider 内使用');
  return value;
}
