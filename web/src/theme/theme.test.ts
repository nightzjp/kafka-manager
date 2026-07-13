import { describe,expect,it } from 'vitest';
import { resolveTheme } from './ThemeProvider';

describe('resolveTheme',()=>{
 it('uses the system preference when mode is system',()=>{
  expect(resolveTheme('system',true)).toBe('dark');
  expect(resolveTheme('system',false)).toBe('light');
 });
 it('keeps an explicit theme',()=>{
  expect(resolveTheme('light',true)).toBe('light');
  expect(resolveTheme('dark',false)).toBe('dark');
 });
});
