import { describe, expect, it } from 'vitest';
import { confirmationMatches } from './confirmation-model';

describe('confirmationMatches', () => {
  it('requires an exact resource-name match when confirmation text is configured', () => {
    expect(confirmationMatches('orders.v1', 'orders.v1')).toBe(true);
    expect(confirmationMatches('orders.v1 ', 'orders.v1')).toBe(false);
    expect(confirmationMatches('ORDERS.V1', 'orders.v1')).toBe(false);
  });

  it('allows a button-only confirmation when no confirmation text is configured', () => {
    expect(confirmationMatches('', undefined)).toBe(true);
  });
});
