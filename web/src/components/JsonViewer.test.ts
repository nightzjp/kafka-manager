import { describe,expect,it } from 'vitest';
import { parseMessageValue,formatJson } from './JsonViewer';

describe('message value parsing',()=>{
 it('recognizes and formats JSON',()=>{
  const result=parseMessageValue('{"order":{"id":7},"ok":true}');
  expect(result.kind).toBe('json');
 expect(formatJson(result.value)).toContain('\n  "order"');
 });
 it('preserves JSON escaping in strings and object keys',()=>{
  const result=parseMessageValue('{"a\\\\b":"line\\n\\\"quoted\\\""}');
  expect(result.kind).toBe('json');
  expect(formatJson(result.value)).toContain('"a\\\\b"');
  expect(formatJson(result.value)).toContain('line\\n\\\"quoted\\\"');
 });
 it('keeps plain text intact',()=>{
  const result=parseMessageValue('not-json');
  expect(result).toEqual({kind:'text',value:'not-json'});
 });
});
