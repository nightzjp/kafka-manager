import { describe, expect, it } from 'vitest';
import { RequestSequence } from './request-sequence';

describe('RequestSequence', () => {
  it('accepts only the most recently started request', () => {
    const sequence = new RequestSequence();
    const older = sequence.begin();
    const newer = sequence.begin();

    expect(sequence.isCurrent(older)).toBe(false);
    expect(sequence.isCurrent(newer)).toBe(true);
  });
});
