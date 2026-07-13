const SIDEBAR_COLLAPSED_KEY = 'kafka-manager.sidebar.collapsed';

type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;

export function readSidebarCollapsed(storage?: StorageLike): boolean {
  if (!storage) return false;

  try {
    return storage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true';
  } catch {
    return false;
  }
}

export function writeSidebarCollapsed(storage: StorageLike | undefined, collapsed: boolean): void {
  if (!storage) return;

  try {
    storage.setItem(SIDEBAR_COLLAPSED_KEY, String(collapsed));
  } catch {
    // Storage can be unavailable in private or hardened browser contexts.
  }
}
