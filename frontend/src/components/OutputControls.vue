<script setup lang="ts">
import type { SampleRate } from '../types/api';
import type { Translate } from '../types/ui';

defineProps<{
  t: Translate;
  sampleRate: SampleRate;
  queueCount: number;
  isProcessing: boolean;
  processingStatus: string;
}>();

const emit = defineEmits<{
  'update:sampleRate': [value: SampleRate];
  process: [];
}>();
</script>

<template>
  <section class="module output-module">
    <div class="module-header">
      <h2 class="module-title">{{ t('output.title') }}</h2>
    </div>
    <div class="output-stack">
      <label class="field-block mb-0" for="rate">
        <span class="field-label">{{ t('output.sample_rate') }}</span>
        <select
          id="rate"
          class="form-select control-select"
          :value="sampleRate"
          @change="emit('update:sampleRate', Number(($event.target as HTMLSelectElement).value) as SampleRate)"
        >
          <option :value="44100">44100 Hz</option>
          <option :value="48000">48000 Hz</option>
          <option :value="96000">96000 Hz</option>
        </select>
      </label>
      <div class="status-cluster">
        <div class="queue-count-label">{{ t('status.queue_ready') }} {{ queueCount }}</div>
        <div class="loading" :class="{ 'is-visible': isProcessing }" role="status" aria-live="polite">
          <span class="spinner-border" aria-hidden="true"></span>
          <span>{{ processingStatus }}</span>
        </div>
      </div>
      <button type="button" class="process-btn" :disabled="queueCount === 0 || isProcessing" @click="emit('process')">
        {{ t('actions.process_download') }}
      </button>
    </div>
  </section>
</template>
