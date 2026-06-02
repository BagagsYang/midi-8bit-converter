<script setup lang="ts">
import { computed } from 'vue';
import { maxMasterGainDb, minMasterGainDb, normaliseDecimalInput } from '../lib';
import { useProEligibility } from '../pro';
import type { SampleRate } from '../types/api';
import type { ChannelBusState, QueuedFile, Translate } from '../types/ui';

const props = defineProps<{
  t: Translate;
  sampleRate: SampleRate;
  queuedFiles: QueuedFile[];
  channelBuses: ChannelBusState[];
  masterGainDb: number;
  limiterEnabled: boolean;
  normaliseEnabled: boolean;
  queueCount: number;
  isProcessing: boolean;
  processingStatus: string;
}>();

const emit = defineEmits<{
  'update:sampleRate': [value: SampleRate];
  updateChannelBusVolume: [channel: number, value: number];
  updateChannelBusMute: [channel: number, value: boolean];
  updateChannelBusSolo: [channel: number, value: boolean];
  updateMasterGainDb: [value: number];
  updateLimiterEnabled: [value: boolean];
  updateNormaliseEnabled: [value: boolean];
  process: [];
}>();

const queuedFilesRef = computed(() => props.queuedFiles);
const { availableChannels } = useProEligibility(queuedFilesRef);
const busByChannel = computed(() => {
  const buses = new Map<number, ChannelBusState>();
  props.channelBuses.forEach((bus) => buses.set(bus.channel, bus));
  return buses;
});

function busFor(channel: number): ChannelBusState {
  return busByChannel.value.get(channel) || {
    channel,
    volume: 1,
    mute: false,
    solo: false,
  };
}
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

      <label class="field-block mb-0" for="masterGain">
        <span class="field-label">{{ t('output.master_gain') }}</span>
        <input
          id="masterGain"
          class="form-control readout"
          type="number"
          :min="minMasterGainDb"
          :max="maxMasterGainDb"
          step="0.5"
          :value="masterGainDb.toFixed(1)"
          @change="emit('updateMasterGainDb', normaliseDecimalInput(($event.target as HTMLInputElement).value, minMasterGainDb, maxMasterGainDb))"
        />
      </label>

      <div class="layer-actions">
        <label class="control-switch">
          <input
            class="control-switch-input"
            type="checkbox"
            :checked="limiterEnabled"
            @change="emit('updateLimiterEnabled', ($event.target as HTMLInputElement).checked)"
          />
          <span class="control-switch-label">{{ t('output.limiter') }}</span>
        </label>
        <label class="control-switch">
          <input
            class="control-switch-input"
            type="checkbox"
            :checked="normaliseEnabled"
            @change="emit('updateNormaliseEnabled', ($event.target as HTMLInputElement).checked)"
          />
          <span class="control-switch-label">{{ t('output.normalise') }}</span>
        </label>
      </div>

      <div v-if="availableChannels.length" class="field-block mb-0">
        <span class="field-label">{{ t('output.channel_buses') }}</span>
        <div v-for="channel in availableChannels" :key="channel" class="layer-actions">
          <span class="module-readout">{{ t('pro.channel', { channel }) }}</span>
          <input
            class="readout"
            type="number"
            min="0"
            max="2"
            step="0.01"
            :value="busFor(channel).volume.toFixed(2)"
            @change="emit('updateChannelBusVolume', channel, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0, 2))"
          />
          <label class="module-readout">
            <input
              type="checkbox"
              :checked="busFor(channel).mute"
              @change="emit('updateChannelBusMute', channel, ($event.target as HTMLInputElement).checked)"
            />
            {{ t('output.mute') }}
          </label>
          <label class="module-readout">
            <input
              type="checkbox"
              :checked="busFor(channel).solo"
              @change="emit('updateChannelBusSolo', channel, ($event.target as HTMLInputElement).checked)"
            />
            {{ t('output.solo') }}
          </label>
        </div>
      </div>

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
