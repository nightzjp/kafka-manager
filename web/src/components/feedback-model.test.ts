import { describe, expect, it } from 'vitest';
import { addFeedback, dismissFeedback, FeedbackItem } from './feedback-model';

const item = (id: string): FeedbackItem => ({ id, tone: 'success', title: id, message: '', duration: 4000 });

describe('feedback model', () => {
  it('keeps only the newest four messages', () => {
    const messages = ['1', '2', '3', '4', '5'].reduce((items, id) => addFeedback(items, item(id)), [] as FeedbackItem[]);
    expect(messages.map((message) => message.id)).toEqual(['2', '3', '4', '5']);
  });

  it('dismisses a message without changing the remaining order', () => {
    expect(dismissFeedback([item('1'), item('2'), item('3')], '2').map((message) => message.id)).toEqual(['1', '3']);
  });
});
