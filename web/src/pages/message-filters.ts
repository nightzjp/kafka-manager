export type KeyOperator = 'contains' | 'exact' | 'prefix';
export type ValueOperator = 'contains' | 'exact';
export type JSONOperator = 'eq' | 'neq' | 'contains' | 'exists' | 'gt' | 'gte' | 'lt' | 'lte';

export type JSONFilterCondition = {
  id: string;
  path: string;
  operator: JSONOperator;
  value: string;
};

export type MessageFilters = {
  keyFilter: string;
  keyOperator: KeyOperator;
  valueFilter: string;
  valueOperator: ValueOperator;
  scanLimit: number;
  jsonFilters: JSONFilterCondition[];
};

const defaultFilters = (): MessageFilters => ({
  keyFilter: '', keyOperator: 'contains', valueFilter: '', valueOperator: 'contains', scanLimit: 5000, jsonFilters: [],
});
const keyOperators: KeyOperator[] = ['contains', 'exact', 'prefix'];
const valueOperators: ValueOperator[] = ['contains', 'exact'];
const jsonOperators: JSONOperator[] = ['eq', 'neq', 'contains', 'exists', 'gt', 'gte', 'lt', 'lte'];
const filterParamNames = ['keyFilter', 'keyOperator', 'valueFilter', 'valueOperator', 'scanLimit', 'jsonFilters'];

function serializableJSONFilters(filters: MessageFilters) {
  return filters.jsonFilters.filter((filter) => filter.path.trim()).map(({ path, operator, value }) => ({ path: path.trim(), operator, value }));
}

export function activeFilterCount(filters: MessageFilters): number {
  return Number(Boolean(filters.keyFilter.trim())) + Number(Boolean(filters.valueFilter.trim())) + serializableJSONFilters(filters).length;
}

export function filterQueryParams(filters: MessageFilters): Record<string, string> {
  const output: Record<string, string> = {};
  const key = filters.keyFilter.trim();
  const value = filters.valueFilter.trim();
  const jsonFilters = serializableJSONFilters(filters);
  if (key) { output.keyFilter = key; output.keyOperator = filters.keyOperator; }
  if (value) { output.valueFilter = value; output.valueOperator = filters.valueOperator; }
  if (jsonFilters.length) output.jsonFilters = JSON.stringify(jsonFilters);
  if (key || value || jsonFilters.length) output.scanLimit = String(filters.scanLimit);
  return output;
}

export function parseMessageFilters(params: URLSearchParams): MessageFilters {
  const output = defaultFilters();
  output.keyFilter = params.get('keyFilter') || '';
  output.valueFilter = params.get('valueFilter') || '';
  const keyOperator = params.get('keyOperator') as KeyOperator | null;
  const valueOperator = params.get('valueOperator') as ValueOperator | null;
  if (keyOperator && keyOperators.includes(keyOperator)) output.keyOperator = keyOperator;
  if (valueOperator && valueOperators.includes(valueOperator)) output.valueOperator = valueOperator;

  const scanLimit = Number(params.get('scanLimit'));
  if (Number.isFinite(scanLimit) && scanLimit > 0) output.scanLimit = Math.min(50000, Math.max(1, Math.floor(scanLimit)));

  try {
    const parsed: unknown = JSON.parse(params.get('jsonFilters') || '[]');
    if (Array.isArray(parsed)) {
      output.jsonFilters = parsed.slice(0, 5).flatMap((item, index) => {
        if (!item || typeof item !== 'object') return [];
        const candidate = item as Record<string, unknown>;
        const operator = candidate.operator as JSONOperator;
        if (typeof candidate.path !== 'string' || !jsonOperators.includes(operator)) return [];
        return [{ id: `url-${index}`, path: candidate.path, operator, value: typeof candidate.value === 'string' ? candidate.value : '' }];
      });
    }
  } catch {
    // A manually edited or stale URL should fall back to an empty JSON filter list.
  }
  return output;
}

export function replaceMessageFilterParams(current: URLSearchParams, filters: MessageFilters): URLSearchParams {
  const next = new URLSearchParams(current);
  filterParamNames.forEach((name) => next.delete(name));
  Object.entries(filterQueryParams(filters)).forEach(([name, value]) => next.set(name, value));
  return next;
}
