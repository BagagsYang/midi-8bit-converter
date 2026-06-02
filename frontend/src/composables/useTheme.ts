import { ref } from 'vue';

type ThemeChoice = 'system' | 'light' | 'dark';

const themeStorageKey = 'octabitTheme';

function systemTheme(): 'light' | 'dark' {
  if (window.matchMedia?.('(prefers-color-scheme: light)').matches) {
    return 'light';
  }
  return 'dark';
}

function initialThemeChoice(): ThemeChoice {
  const stored = localStorage.getItem(themeStorageKey);
  return stored === 'light' || stored === 'dark' ? stored : 'system';
}

export function useTheme() {
  const themeChoice = ref<ThemeChoice>(initialThemeChoice());
  const activeTheme = ref<'light' | 'dark'>('dark');

  function applyThemeChoice(choice: ThemeChoice) {
    themeChoice.value = choice;
    if (choice === 'system') {
      localStorage.removeItem(themeStorageKey);
      activeTheme.value = systemTheme();
    } else {
      localStorage.setItem(themeStorageKey, choice);
      activeTheme.value = choice;
    }
    document.documentElement.classList.add('theme-change-instant');
    document.documentElement.setAttribute('data-bs-theme', activeTheme.value);
    void document.documentElement.offsetHeight;
    document.documentElement.classList.remove('theme-change-instant');
  }

  function syncSystemTheme() {
    if (themeChoice.value === 'system') {
      applyThemeChoice('system');
    }
  }

  return {
    themeChoice,
    activeTheme,
    applyThemeChoice,
    syncSystemTheme,
  };
}
