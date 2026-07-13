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
