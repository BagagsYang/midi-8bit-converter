<script setup lang="ts">
import { ref } from 'vue';
import { iconSvg } from '../icons';
import { formatFileSize } from '../lib';
import type { QueuedFile, Translate } from '../types/ui';

defineProps<{
  t: Translate;
  files: QueuedFile[];
  keepQueue: boolean;
}>();

const emit = defineEmits<{
  addFiles: [files: FileList | File[]];
  removeFile: [index: number];
  clearQueue: [];
  reorder: [fromIndex: number, toIndex: number];
  'update:keepQueue': [value: boolean];
}>();

const dragStartIndex = ref<number | null>(null);

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  if (input.files?.length) {
    emit('addFiles', input.files);
  }
  input.value = '';
}

function onDropFiles(event: DragEvent) {
  event.preventDefault();
  const target = event.currentTarget as HTMLElement;
  target.classList.remove('dragover');
  if (event.dataTransfer?.files.length) {
    emit('addFiles', event.dataTransfer.files);
  }
}
</script>

<template>
  <section class="module upload-module">
    <div class="module-header">
      <h2 class="module-title">{{ t('files.upload') }}</h2>
    </div>
    <div
      class="drop-zone"
      @dragover.prevent="($event.currentTarget as HTMLElement).classList.add('dragover')"
      @dragleave="($event.currentTarget as HTMLElement).classList.remove('dragover')"
      @dragend="($event.currentTarget as HTMLElement).classList.remove('dragover')"
      @drop="onDropFiles"
    >
      <span class="drop-zone-prompt">
        <span>{{ t('queue.drop_prompt') }}</span>
        <span class="drop-zone-hint">{{ t('queue.drop_hint') }}</span>
      </span>
      <input class="form-control" type="file" accept=".mid,.midi" multiple @change="onFileChange" />
    </div>
    <div class="module-divider"></div>
    <div class="module-header">
      <h2 class="module-title">{{ t('queue.title') }}</h2>
      <span class="module-readout">{{ files.length }}</span>
    </div>
    <p v-if="files.length === 0" class="empty-state">{{ t('queue.empty') }}</p>
    <ul v-else class="panel-list queue-list">
      <li
        v-for="(file, index) in files"
        :key="file.fileId"
        class="queue-item"
        draggable="true"
        :data-index="index"
        :data-full-name="file.name"
        @dragstart="dragStartIndex = index; ($event.currentTarget as HTMLElement).classList.add('dragging')"
        @dragend="($event.currentTarget as HTMLElement).classList.remove('dragging')"
        @dragover.prevent="($event.currentTarget as HTMLElement).classList.add('drag-over')"
        @dragleave="($event.currentTarget as HTMLElement).classList.remove('drag-over')"
        @drop.prevent="($event.currentTarget as HTMLElement).classList.remove('drag-over'); dragStartIndex !== null && emit('reorder', dragStartIndex, index)"
      >
        <div class="min-w-0">
          <div class="file-name">{{ file.name }}</div>
          <div class="file-meta">{{ formatFileSize(file.size) }}</div>
        </div>
        <button type="button" class="remove-btn" :aria-label="t('queue.remove_file', { filename: file.name })" @click="emit('removeFile', index)" v-html="iconSvg('x')"></button>
      </li>
    </ul>
    <button v-if="files.length > 0" type="button" class="utility-btn w-100 mt-3" @click="emit('clearQueue')">
      {{ t('queue.clear') }}
    </button>
    <div class="control-switch">
      <input
        class="control-switch-input"
        type="checkbox"
        id="keepQueueToggle"
        :checked="keepQueue"
        @change="emit('update:keepQueue', ($event.target as HTMLInputElement).checked)"
      />
      <label class="control-switch-label" for="keepQueueToggle">
        <span class="control-switch-track" aria-hidden="true">
          <span class="control-switch-thumb"></span>
        </span>
        <span class="control-switch-text">{{ t('queue.keep_after') }}</span>
      </label>
    </div>
  </section>
</template>
