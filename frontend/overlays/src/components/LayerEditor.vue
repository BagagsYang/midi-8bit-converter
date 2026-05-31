<script setup lang="ts">
import { computed } from 'vue';
import { iconSvg } from '../icons';
import {
  maxLayerOctaveShift,
  maxLayers,
  minLayerOctaveShift,
  normaliseDecimalInput,
  standardWaveTypeOptions,
  waveTypeOptions,
} from '../lib';
import { proMode, useProEligibility } from '../pro';
import type { WaveType } from '../types/api';
import type { LayerState, QueuedFile, Translate } from '../types/ui';
import FrequencyCurveEditor from './FrequencyCurveEditor.vue';

const props = defineProps<{
  t: Translate;
  layers: LayerState[];
  layerCount: number;
  queuedFiles: QueuedFile[];
}>();

const emit = defineEmits<{
  updateLayerType: [layerIndex: number, value: WaveType];
  updateLayerDuty: [layerIndex: number, value: number];
  updateLayerVolume: [layerIndex: number, value: number];
  updateLayerMidiChannels: [layerIndex: number, value: number[]];
  updateLayerVibratoDepth: [layerIndex: number, value: number];
  updateLayerVibratoRate: [layerIndex: number, value: number];
  updateLayerOctaveShift: [layerIndex: number, delta: number];
  toggleCurve: [layerIndex: number, enabled: boolean];
  addCurvePoint: [layerIndex: number];
  removeSelectedPoint: [layerIndex: number];
  resetCurve: [layerIndex: number];
  selectCurvePoint: [layerIndex: number, pointIndex: number];
  startCurvePointDrag: [layerIndex: number, pointIndex: number, event: PointerEvent];
  addLayer: [];
  removeLayer: [];
  resetLayers: [];
  playPreview: [layerIndex: number];
}>();

const activeLayers = computed(() => props.layers.slice(0, props.layerCount));
const queuedFilesRef = computed(() => props.queuedFiles);
const { hasQueuedFiles, allQueuedFilesMultiChannel, availableChannels } = useProEligibility(queuedFilesRef);
const activeTypes = computed(() => new Set(activeLayers.value.map((layer) => layer.type)));
const visibleWaveTypeOptions = computed(() => (proMode.value ? waveTypeOptions : standardWaveTypeOptions));
const canUseProControls = computed(() => proMode.value && hasQueuedFiles.value && allQueuedFilesMultiChannel.value);
const canAddLayer = computed(() => (
  props.layerCount < maxLayers
  && (proMode.value || standardWaveTypeOptions.some(([value]) => !activeTypes.value.has(value)))
));

function faderFillPercent(value: number, min: number, max: number): string {
  const ratio = (Number(value) - min) / (max - min);
  return `${Math.min(Math.max(ratio * 100, 0), 100).toFixed(2)}%`;
}

function faderScale(ticks: Array<{ value: number; label: string }>, min: number, max: number) {
  return ticks.map((tick, index) => ({
    ...tick,
    position: faderFillPercent(tick.value, min, max),
    edgeClass: index === 0 ? 'is-start' : index === ticks.length - 1 ? 'is-end' : '',
  }));
}

function isWaveDisabled(value: WaveType, layerIndex: number): boolean {
  if (value === 'noise') {
    return !canUseProControls.value;
  }
  return !proMode.value && props.layers.slice(0, props.layerCount).some((layer, index) => index !== layerIndex && layer.type === value);
}

function selectedChannelValues(event: Event): number[] {
  const select = event.target as HTMLSelectElement;
  return Array.from(select.selectedOptions).map((option) => Number(option.value));
}

function formatOctaveShift(value: number): string {
  if (value > 0) return `+${value}`;
  return String(value);
}

function proHint(): string {
  if (!hasQueuedFiles.value) return props.t('pro.requires_queue');
  if (!allQueuedFilesMultiChannel.value) return props.t('pro.requires_multi_channel');
  return props.t('pro.multi_channel_ready');
}
</script>

<template>
  <section class="module">
    <div class="module-header">
      <h2 class="module-title">{{ t('parameters.title') }}</h2>
      <div class="module-header-actions">
        <button type="button" class="module-readout module-reset-btn" @click="emit('resetLayers')">
          {{ t('parameters.reset') }}
        </button>
        <span class="module-readout">{{ t('parameters.readout') }}</span>
      </div>
    </div>

    <div>
      <div v-for="(layer, layerIndex) in activeLayers" :key="layerIndex" class="layer-card">
        <div class="layer-title-row">
          <div>
            <div class="layer-title">{{ t('layer.title', { index: layerIndex + 1 }) }}</div>
          </div>
          <button
            type="button"
            class="preview-btn"
            :disabled="layer.type === 'noise'"
            :title="t('layer.play_preview')"
            :aria-label="t('layer.play_preview')"
            @click="emit('playPreview', layerIndex)"
            v-html="iconSvg('play')"
          ></button>
        </div>

        <div class="layer-control-grid">
          <div class="field-block waveform-field">
            <label class="field-label" :for="`waveType${layerIndex}`">{{ t('layer.waveform_type') }}</label>
            <select
              class="form-select control-select"
              :id="`waveType${layerIndex}`"
              :value="layer.type"
              @change="emit('updateLayerType', layerIndex, ($event.target as HTMLSelectElement).value as WaveType)"
            >
              <option
                v-for="[value, labelKey] in visibleWaveTypeOptions"
                :key="value"
                :value="value"
                :disabled="isWaveDisabled(value, layerIndex)"
              >
                {{ t(labelKey) }}
              </option>
            </select>
          </div>

          <div class="field-block" :style="{ display: layer.type === 'pulse' ? 'grid' : 'none' }">
            <label class="fader-label" :for="`dutyFader${layerIndex}`">
              <span>{{ t('layer.pulse_width') }}</span>
              <input
                type="number"
                class="readout"
                :id="`dutyValue${layerIndex}`"
                min="0.01"
                max="0.99"
                step="0.01"
                :value="layer.duty.toFixed(2)"
                inputmode="decimal"
                @change="emit('updateLayerDuty', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0.01, 0.99))"
              />
            </label>
            <div class="fader-shell">
              <input
                type="range"
                class="fader-input"
                :id="`dutyFader${layerIndex}`"
                min="0.01"
                max="0.99"
                step="0.01"
                :value="layer.duty"
                :style="{ '--fill': faderFillPercent(layer.duty, 0.01, 0.99) }"
                @input="emit('updateLayerDuty', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0.01, 0.99))"
              />
            </div>
            <div class="fader-scale" aria-hidden="true">
              <template v-for="tick in faderScale([
                { value: 0.01, label: '0.01' },
                { value: 0.25, label: '0.25' },
                { value: 0.50, label: '0.50' },
                { value: 0.75, label: '0.75' },
                { value: 0.99, label: '0.99' },
              ], 0.01, 0.99)" :key="tick.label">
                <span class="fader-scale-mark" :class="tick.edgeClass" :style="{ '--tick-position': tick.position }"></span>
                <span class="fader-scale-label" :class="tick.edgeClass" :style="{ '--tick-position': tick.position }">{{ tick.label }}</span>
              </template>
            </div>
          </div>

          <div class="field-block layer-volume-control" :class="{ 'layer-volume-wide': layer.type !== 'pulse' }">
            <label class="fader-label" :for="`volumeFader${layerIndex}`">
              <span>{{ t('layer.base_volume') }}</span>
              <input
                type="number"
                class="readout"
                :id="`volumeValue${layerIndex}`"
                min="0.00"
                max="2.00"
                step="0.01"
                :value="layer.volume.toFixed(2)"
                inputmode="decimal"
                @change="emit('updateLayerVolume', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0.0, 2.0))"
              />
            </label>
            <div class="fader-shell">
              <input
                type="range"
                class="fader-input"
                :id="`volumeFader${layerIndex}`"
                min="0.0"
                max="2.0"
                step="0.01"
                :value="layer.volume"
                :style="{ '--fill': faderFillPercent(layer.volume, 0.0, 2.0) }"
                @input="emit('updateLayerVolume', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0.0, 2.0))"
              />
            </div>
            <div class="fader-scale" aria-hidden="true">
              <template v-for="tick in faderScale([
                { value: 0.00, label: '0.00' },
                { value: 0.50, label: '0.50' },
                { value: 1.00, label: '1.00' },
                { value: 1.50, label: '1.50' },
                { value: 2.00, label: '2.00' },
              ], 0.0, 2.0)" :key="tick.label">
                <span class="fader-scale-mark" :class="tick.edgeClass" :style="{ '--tick-position': tick.position }"></span>
                <span class="fader-scale-label" :class="tick.edgeClass" :style="{ '--tick-position': tick.position }">{{ tick.label }}</span>
              </template>
            </div>
          </div>

          <div class="control-switch layer-curve-toggle">
            <input
              class="control-switch-input"
              type="checkbox"
              :id="`curveToggle${layerIndex}`"
              :checked="layer.curveEnabled"
              @change="emit('toggleCurve', layerIndex, ($event.target as HTMLInputElement).checked)"
            />
            <label class="control-switch-label" :for="`curveToggle${layerIndex}`">
              <span class="control-switch-track" aria-hidden="true">
                <span class="control-switch-thumb"></span>
              </span>
              <span class="control-switch-text">{{ t('layer.enable_curve') }}</span>
            </label>
          </div>
        </div>

        <div
          v-if="proMode"
          class="extension-layer-panel"
          :class="{ 'is-disabled': !canUseProControls }"
        >
          <p class="extension-hint">{{ proHint() }}</p>
          <div class="extension-control-grid">
            <div class="field-block">
              <label class="field-label" :for="`midiChannels${layerIndex}`">{{ t('pro.midi_channels') }}</label>
              <select
                class="form-select control-select"
                :id="`midiChannels${layerIndex}`"
                multiple
                :disabled="!canUseProControls"
                @change="emit('updateLayerMidiChannels', layerIndex, selectedChannelValues($event))"
              >
                <option
                  v-for="channel in availableChannels"
                  :key="channel"
                  :value="channel"
                  :selected="layer.midiChannels.includes(channel)"
                >
                  {{ t('pro.channel', { channel }) }}
                </option>
              </select>
            </div>

            <div class="field-block">
              <div class="field-label">
                <span>{{ t('pro.octave_shift') }}</span>
                <span class="module-readout">{{ formatOctaveShift(layer.octaveShift) }}</span>
              </div>
              <div class="layer-actions">
                <button
                  type="button"
                  class="utility-btn fw-bold"
                  :disabled="!canUseProControls || layer.octaveShift <= minLayerOctaveShift"
                  :aria-label="t('pro.octave_down_aria')"
                  @click="emit('updateLayerOctaveShift', layerIndex, -1)"
                >
                  {{ t('pro.octave_down') }}
                </button>
                <button
                  type="button"
                  class="utility-btn fw-bold"
                  :disabled="!canUseProControls || layer.octaveShift >= maxLayerOctaveShift"
                  :aria-label="t('pro.octave_up_aria')"
                  @click="emit('updateLayerOctaveShift', layerIndex, 1)"
                >
                  {{ t('pro.octave_up') }}
                </button>
              </div>
            </div>

            <div class="field-block">
              <label class="fader-label" :for="`vibratoDepth${layerIndex}`">
                <span>{{ t('pro.vibrato_depth') }}</span>
                <input
                  type="number"
                  class="readout"
                  :id="`vibratoDepthValue${layerIndex}`"
                  min="0"
                  max="200"
                  step="1"
                  :value="layer.vibratoDepthCents.toFixed(0)"
                  inputmode="decimal"
                  :disabled="!canUseProControls"
                  @change="emit('updateLayerVibratoDepth', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0, 200))"
                />
              </label>
              <div class="fader-shell">
                <input
                  type="range"
                  class="fader-input"
                  :id="`vibratoDepth${layerIndex}`"
                  min="0"
                  max="200"
                  step="1"
                  :value="layer.vibratoDepthCents"
                  :disabled="!canUseProControls"
                  :style="{ '--fill': faderFillPercent(layer.vibratoDepthCents, 0, 200) }"
                  @input="emit('updateLayerVibratoDepth', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0, 200))"
                />
              </div>
            </div>

            <div class="field-block">
              <label class="fader-label" :for="`vibratoRate${layerIndex}`">
                <span>{{ t('pro.vibrato_rate') }}</span>
                <input
                  type="number"
                  class="readout"
                  :id="`vibratoRateValue${layerIndex}`"
                  min="0.1"
                  max="20"
                  step="0.1"
                  :value="layer.vibratoRateHz.toFixed(1)"
                  inputmode="decimal"
                  :disabled="!canUseProControls || layer.vibratoDepthCents <= 0"
                  @change="emit('updateLayerVibratoRate', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0.1, 20))"
                />
              </label>
              <div class="fader-shell">
                <input
                  type="range"
                  class="fader-input"
                  :id="`vibratoRate${layerIndex}`"
                  min="0.1"
                  max="20"
                  step="0.1"
                  :value="layer.vibratoRateHz"
                  :disabled="!canUseProControls || layer.vibratoDepthCents <= 0"
                  :style="{ '--fill': faderFillPercent(layer.vibratoRateHz, 0.1, 20) }"
                  @input="emit('updateLayerVibratoRate', layerIndex, normaliseDecimalInput(($event.target as HTMLInputElement).value, 0.1, 20))"
                />
              </div>
            </div>
          </div>
        </div>

        <FrequencyCurveEditor
          v-if="layer.curveEnabled"
          :t="t"
          :layer="layer"
          :layer-index="layerIndex"
          :can-remove-selected="layer.frequencyCurve.length > 2 && layer.selectedPointIndex > 0 && layer.selectedPointIndex < layer.frequencyCurve.length - 1"
          @add-point="emit('addCurvePoint', layerIndex)"
          @remove-selected="emit('removeSelectedPoint', layerIndex)"
          @reset="emit('resetCurve', layerIndex)"
          @select-point="emit('selectCurvePoint', layerIndex, $event)"
          @start-point-drag="(pointIndex, event) => emit('startCurvePointDrag', layerIndex, pointIndex, event)"
        />
      </div>
    </div>

    <div class="layer-actions">
      <button type="button" class="utility-btn fw-bold" :disabled="!canAddLayer" @click="emit('addLayer')">
        {{ t('layers.add') }}
      </button>
      <button type="button" class="utility-btn fw-bold" :disabled="layerCount === 1" @click="emit('removeLayer')">
        {{ t('layers.remove') }}
      </button>
    </div>
  </section>
</template>
