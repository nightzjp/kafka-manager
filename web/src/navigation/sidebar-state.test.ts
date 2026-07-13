import { describe, expect, it, vi } from 'vitest';
import { readSidebarCollapsed, writeSidebarCollapsed } from './sidebar-state';

describe('sidebar preference', () => {
  it('defaults to expanded and only accepts an explicit true value', () => {
    expect(readSidebarCollapsed(undefined)).toBe(false);
    expect(readSidebarCollapsed({ getItem: () => null, setItem: () => {} })).toBe(false);
    expect(readSidebarCollapsed({ getItem: () => 'false', setItem: () => {} })).toBe(false);
    expect(readSidebarCollapsed({ getItem: () => 'true', setItem: () => {} })).toBe(true);
  });

  it('persists the collapsed state under the stable application key', () => {
    const setItem = vi.fn();
    writeSidebarCollapsed({ getItem: () => null, setItem }, true);
    expect(setItem).toHaveBeenCalledWith('kafka-manager.sidebar.collapsed', 'true');
  });

  it('does not crash when browser storage is unavailable', () => {
    const storage = { getItem: () => { throw new Error('blocked'); }, setItem: () => { throw new Error('blocked'); } };
    expect(readSidebarCollapsed(storage)).toBe(false);
    expect(() => writeSidebarCollapsed(storage, true)).not.toThrow();
  });
});
