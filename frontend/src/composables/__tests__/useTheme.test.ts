import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useTheme } from '../useTheme';

describe('useTheme', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('defaults to system when no stored preference', () => {
    const { themeChoice } = useTheme();
    expect(themeChoice.value).toBe('system');
  });

  it('restores stored preference', () => {
    localStorage.setItem('octabitTheme', 'dark');
    const { themeChoice } = useTheme();
    expect(themeChoice.value).toBe('dark');
  });

  it('applyThemeChoice sets active theme and persists', () => {
    const { activeTheme, applyThemeChoice } = useTheme();
    applyThemeChoice('light');
    expect(activeTheme.value).toBe('light');
    expect(localStorage.getItem('octabitTheme')).toBe('light');
  });

  it('applyThemeChoice system removes storage and uses system preference', () => {
    localStorage.setItem('octabitTheme', 'dark');
    const { activeTheme, applyThemeChoice } = useTheme();
    applyThemeChoice('system');
    expect(localStorage.getItem('octabitTheme')).toBeNull();
  });

  it('sets data-bs-theme attribute on document', () => {
    const { applyThemeChoice } = useTheme();
    applyThemeChoice('dark');
    expect(document.documentElement.getAttribute('data-bs-theme')).toBe('dark');
  });
});
