import { describe, expect, it, vi } from 'vitest';
import { errorBoundaryState, reloadApplication } from './AppErrorBoundary';

describe('AppErrorBoundary', () => {
  it('converts a render error into a recoverable failed state', () => {
    expect(errorBoundaryState(new Error('partition metadata is missing'))).toEqual({
      failed: true,
      message: 'partition metadata is missing',
    });
  });

  it('falls back to a safe message for non-error values', () => {
    expect(errorBoundaryState('broken')).toEqual({
      failed: true,
      message: '页面渲染遇到未知异常',
    });
  });

  it('reloads the application when the user retries', () => {
    const reload = vi.fn();
    reloadApplication(reload);
    expect(reload).toHaveBeenCalledOnce();
  });
});
