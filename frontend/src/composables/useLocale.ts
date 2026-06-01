import { computed, ref } from 'vue';
import en from '../i18n/en.json';
import es from '../i18n/es.json';
import fr from '../i18n/fr.json';
import ja from '../i18n/ja.json';
import zhCn from '../i18n/zh-CN.json';
import zhTw from '../i18n/zh-TW.json';

type Locale = 'en' | 'es' | 'fr' | 'ja' | 'zh-CN' | 'zh-TW';

const translationsByLocale: Record<Locale, Record<string, string>> = {
  en,
  es,
  fr,
  ja,
  'zh-CN': zhCn,
  'zh-TW': zhTw,
};
const supportedLocales: Locale[] = ['en', 'es', 'fr', 'ja', 'zh-CN', 'zh-TW'];
const defaultLocale: Locale = 'en';
const localeCookieName = 'web_locale';

function normaliseLocale(value: string | null): Locale {
  return supportedLocales.includes(value as Locale) ? value as Locale : defaultLocale;
}

function initialLocale(): Locale {
  const urlLocale = new URLSearchParams(window.location.search).get('lang');
  const cookieLocale = document.cookie
    .split('; ')
    .find((part) => part.startsWith(`${localeCookieName}=`))
    ?.split('=')[1];
  return normaliseLocale(urlLocale || (cookieLocale ? decodeURIComponent(cookieLocale) : null));
}

export function useLocale() {
  const locale = ref<Locale>(initialLocale());

  const translations = computed(() => translationsByLocale[locale.value] || translationsByLocale[defaultLocale]);

  function t(key: string, params: Record<string, string | number> = {}): string {
    const template = Object.prototype.hasOwnProperty.call(translations.value, key) ? translations.value[key] : key;
    return template.replace(/\{(\w+)\}/g, (match, token) => (
      Object.prototype.hasOwnProperty.call(params, token) ? String(params[token]) : match
    ));
  }

  function updateLocale(value: string) {
    const nextLocale = normaliseLocale(value);
    locale.value = nextLocale;
    document.documentElement.lang = nextLocale;
    document.title = t('meta.site_title');
    document.cookie = `${localeCookieName}=${encodeURIComponent(nextLocale)}; path=/; max-age=${60 * 60 * 24 * 365}; SameSite=Lax`;
    const url = new URL(window.location.href);
    if (nextLocale === defaultLocale) {
      url.searchParams.delete('lang');
    } else {
      url.searchParams.set('lang', nextLocale);
    }
    window.history.replaceState({}, '', url.toString());
  }

  return {
    locale,
    t,
    supportedLocales: supportedLocales as unknown as string[],
    updateLocale,
  };
}
