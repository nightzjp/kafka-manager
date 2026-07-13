import { describe, expect, it } from 'vitest';
import { activeFilterCount, filterQueryParams, MessageFilters } from './message-filters';

const base = (): MessageFilters => ({
  keyFilter: '', keyOperator: 'contains', valueFilter: '', valueOperator: 'contains', scanLimit: 5000, jsonFilters: [],
});

describe('message filter query parameters', () => {
  it('serializes active Key, Value and complete JSON filters', () => {
    const filters: MessageFilters = {
      keyFilter: ' order-', keyOperator: 'prefix', valueFilter: 'SUCCESS', valueOperator: 'contains', scanLimit: 12000,
      jsonFilters: [
        { id: '1', path: ' data.user.id ', operator: 'eq', value: '10086' },
        { id: '2', path: '', operator: 'contains', value: 'ignored' },
        { id: '3', path: 'deletedAt', operator: 'exists', value: '' },
      ],
    };
    const params = filterQueryParams(filters);
    expect(params.keyFilter).toBe('order-');
    expect(params.keyOperator).toBe('prefix');
    expect(params.valueFilter).toBe('SUCCESS');
    expect(params.scanLimit).toBe('12000');
    expect(JSON.parse(params.jsonFilters)).toEqual([
      { path: 'data.user.id', operator: 'eq', value: '10086' },
      { path: 'deletedAt', operator: 'exists', value: '' },
    ]);
    expect(activeFilterCount(filters)).toBe(4);
  });

  it('omits every filter parameter when no filters are active', () => {
    expect(filterQueryParams(base())).toEqual({});
    expect(activeFilterCount(base())).toBe(0);
  });
});
