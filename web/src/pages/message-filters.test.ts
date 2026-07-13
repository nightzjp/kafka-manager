import { describe, expect, it } from 'vitest';
import { activeFilterCount, filterQueryParams, MessageFilters, parseMessageFilters, replaceMessageFilterParams } from './message-filters';

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

  it('restores valid filters from a shared URL and clamps the scan limit', () => {
    const params = new URLSearchParams({
      keyFilter: 'device-', keyOperator: 'prefix', valueFilter: 'online', valueOperator: 'exact', scanLimit: '90000',
      jsonFilters: JSON.stringify([{ path: 'device.sn', operator: 'eq', value: 'SN-1' }]),
    });
    expect(parseMessageFilters(params)).toEqual({
      keyFilter: 'device-', keyOperator: 'prefix', valueFilter: 'online', valueOperator: 'exact', scanLimit: 50000,
      jsonFilters: [{ id: 'url-0', path: 'device.sn', operator: 'eq', value: 'SN-1' }],
    });
  });

  it('uses safe defaults for malformed URL filters', () => {
    const filters = parseMessageFilters(new URLSearchParams({ keyOperator: 'regex', valueOperator: 'bad', scanLimit: 'NaN', jsonFilters: '{bad' }));
    expect(filters).toEqual(base());
  });

  it('replaces only filter parameters and preserves unrelated URL state', () => {
    const current = new URLSearchParams('view=compact&keyFilter=old&scanLimit=3');
    const next = replaceMessageFilterParams(current, { ...base(), keyFilter: 'new', keyOperator: 'exact' });
    expect(next.get('view')).toBe('compact');
    expect(next.get('keyFilter')).toBe('new');
    expect(next.get('keyOperator')).toBe('exact');
    expect(next.get('scanLimit')).toBe('5000');
  });
});
