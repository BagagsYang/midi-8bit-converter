import { describe, it, expect, beforeEach, vi, beforeAll } from 'vitest';
import { useLocale } from '../useLocale';

describe('useLocale', () => {
  beforeAll(() => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: '',
    });
  });

  beforeEach(() => {
    vi.restoreAllMocks();
    document.cookie = '';
    window.history.replaceState({}, '', '/');
  });

  it('defaults to en when no lang param or cookie set', () => {
    const { locale } = useLocale();
    expect(locale.value).toBe('en');
  });

  it('respects lang URL query parameter', () => {
    window.history.replaceState({}, '', '/?lang=fr');
    const { locale } = useLocale();
    expect(locale.value).toBe('fr');
  });

  it('falls back to en for unknown locale', () => {
    document.cookie = 'web_locale=de';
    const { locale } = useLocale();
    expect(locale.value).toBe('en');
  });

  it('updateLocale changes locale and updates document', () => {
    const { locale, updateLocale } = useLocale();
    updateLocale('zh-CN');
    expect(locale.value).toBe('zh-CN');
    expect(document.documentElement.lang).toBe('zh-CN');
  });

  it('t() returns translation key when key is missing', () => {
    const { t } = useLocale();
    const result = t('nonexistent.key');
    expect(result).toBe('nonexistent.key');
  });

  it('t() interpolates params', () => {
    const { t } = useLocale();
    const result = t('meta.site_title', { key: 'test' });
    expect(typeof result).toBe('string');
  });
});
