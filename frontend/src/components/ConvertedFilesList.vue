<script setup lang="ts">
import { formatFileSize } from '../lib';
import type { ConvertedItem, Translate } from '../types/ui';

defineProps<{
  t: Translate;
  files: ConvertedItem[];
}>();

const emit = defineEmits<{
  download: [index: number];
  clear: [];
}>();
</script>

<template>
  <section class="module converted-module">
    <div class="module-header">
      <h2 class="module-title">{{ t('converted.title') }}</h2>
      <span class="module-readout">{{ files.length }}</span>
    </div>
    <p v-if="files.length === 0" class="empty-state">{{ t('converted.empty') }}</p>
    <ul v-else class="panel-list converted-list">
      <li v-for="(file, index) in files" :key="file.jobId" class="converted-item">
        <div class="min-w-0">
          <div class="file-name">{{ file.name }}</div>
          <div class="file-meta">{{ formatFileSize(file.size) }} / {{ file.sourceName }}</div>
        </div>
        <button type="button" class="download-btn" @click="emit('download', index)">
          {{ t('converted.download') }}
        </button>
      </li>
    </ul>
    <button v-if="files.length > 0" type="button" class="utility-btn w-100 mt-3" @click="emit('clear')">
      {{ t('converted.clear') }}
    </button>
  </section>
</template>
