import { describe, it, expect, beforeEach, vi, beforeAll } from 'vitest';
import en from '../../i18n/en.json';
import es from '../../i18n/es.json';
import fr from '../../i18n/fr.json';
import ja from '../../i18n/ja.json';
import zhCn from '../../i18n/zh-CN.json';
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

  it('respects Spanish lang URL query parameter', () => {
    window.history.replaceState({}, '', '/?lang=es');
    const { locale } = useLocale();
    expect(locale.value).toBe('es');
  });

  it('respects Japanese lang URL query parameter', () => {
    window.history.replaceState({}, '', '/?lang=ja');
    const { locale } = useLocale();
    expect(locale.value).toBe('ja');
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

  it('updateLocale persists Spanish locale in document, cookie, and URL', () => {
    const { locale, updateLocale } = useLocale();
    updateLocale('es');
    expect(locale.value).toBe('es');
    expect(document.documentElement.lang).toBe('es');
    expect(document.cookie).toContain('web_locale=es');
    expect(window.location.search).toBe('?lang=es');
  });

  it('updateLocale persists Japanese locale in document, cookie, and URL', () => {
    const { locale, updateLocale } = useLocale();
    updateLocale('ja');
    expect(locale.value).toBe('ja');
    expect(document.documentElement.lang).toBe('ja');
    expect(document.cookie).toContain('web_locale=ja');
    expect(window.location.search).toBe('?lang=ja');
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

  it('keeps frontend catalog key sets aligned', () => {
    const expectedKeys = Object.keys(en).sort();
    for (const catalog of [es, fr, ja, zhCn]) {
      expect(Object.keys(catalog).sort()).toEqual(expectedKeys);
    }
  });
});
