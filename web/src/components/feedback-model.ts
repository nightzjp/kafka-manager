export type FeedbackTone = 'success' | 'error' | 'info';

export type FeedbackItem = {
  id: string;
  tone: FeedbackTone;
  title: string;
  message: string;
  duration: number;
};

export function addFeedback(items: FeedbackItem[], item: FeedbackItem): FeedbackItem[] {
  return [...items, item].slice(-4);
}

export function dismissFeedback(items: FeedbackItem[], id: string): FeedbackItem[] {
  return items.filter((item) => item.id !== id);
}
