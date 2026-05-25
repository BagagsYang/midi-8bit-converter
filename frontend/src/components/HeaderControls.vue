<script setup lang="ts">
import { computed } from 'vue';
import { iconSvg } from '../icons';
import type { Translate } from '../types/ui';

type ThemeChoice = 'system' | 'light' | 'dark';

const props = defineProps<{
  t: Translate;
  locale: string;
  supportedLocales: string[];
  themeChoice: ThemeChoice;
  activeTheme: 'light' | 'dark';
}>();

const emit = defineEmits<{
  'update:locale': [value: string];
  'update:themeChoice': [value: ThemeChoice];
}>();

const themeIcon = computed(() => {
  if (props.themeChoice === 'light') return iconSvg('sun', 'lucide-icon theme-option-icon');
  if (props.themeChoice === 'dark') return iconSvg('moon-star', 'lucide-icon theme-option-icon');
  return iconSvg(props.activeTheme === 'light' ? 'sun' : 'moon-star', 'lucide-icon theme-option-icon');
});
</script>

<template>
  <header class="control-header">
    <div class="brand-lockup">
      <div class="logo-mark" aria-hidden="true">
        <div class="logo-bars">
          <span></span>
          <span></span>
          <span></span>
          <span></span>
        </div>
      </div>
      <div>
        <h1 class="brand-title">{{ t('meta.page_title') }}</h1>
        <div class="brand-subtitle">{{ t('meta.subtitle') }}</div>
      </div>
    </div>
    <div class="top-controls">
      <span class="theme-select-frame">
        <span class="theme-select-icon" aria-hidden="true" v-html="themeIcon"></span>
        <select
          class="form-select form-select-sm theme-select"
          :value="themeChoice"
          :title="t('settings.theme')"
          :aria-label="t('settings.theme')"
          @change="emit('update:themeChoice', ($event.target as HTMLSelectElement).value as ThemeChoice)"
        >
          <option value="system">{{ t('settings.theme_system') }}</option>
          <option value="light">{{ t('settings.theme_light') }}</option>
          <option value="dark">{{ t('settings.theme_dark') }}</option>
        </select>
      </span>
      <span class="language-select-frame">
        <span class="language-select-icon" aria-hidden="true" v-html="iconSvg('languages', 'lucide-icon language-option-icon')"></span>
        <select
          class="form-select form-select-sm language-select"
          :value="locale"
          :title="t('toolbar.language_title')"
          :aria-label="t('toolbar.language_title')"
          @change="emit('update:locale', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="languageCode in supportedLocales" :key="languageCode" :value="languageCode">
            {{ t(`toolbar.language_option.${languageCode}`) }}
          </option>
        </select>
      </span>
      <a
        class="github-link"
        href="https://github.com/bagags/octabit"
        target="_blank"
        rel="noopener noreferrer"
        :title="t('toolbar.github_repo')"
        :aria-label="t('toolbar.github_repo')"
      >
        <span class="github-link-icon" aria-hidden="true" v-html="iconSvg('github', 'hugeicon github-link-svg')"></span>
      </a>
    </div>
  </header>
</template>
