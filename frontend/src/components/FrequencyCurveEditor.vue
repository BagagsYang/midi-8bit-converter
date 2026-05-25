<script setup lang="ts">
import { computed } from 'vue';
import {
  formatFrequency,
  formatGainDb,
  maxCurveFrequencyHz,
  maxCurveGainDb,
  minCurveFrequencyHz,
  minCurveGainDb,
} from '../lib';
import type { LayerState, Translate } from '../types/ui';

const props = defineProps<{
  t: Translate;
  layer: LayerState;
  layerIndex: number;
  canRemoveSelected: boolean;
}>();

const emit = defineEmits<{
  addPoint: [];
  removeSelected: [];
  reset: [];
  selectPoint: [pointIndex: number];
  startPointDrag: [pointIndex: number, event: PointerEvent];
}>();

const curveWidth = 320;
const curveHeight = 150;
const margin = { top: 14, right: 14, bottom: 24, left: 38 };
const curveLogSpan = Math.log(maxCurveFrequencyHz) - Math.log(minCurveFrequencyHz);

const plotWidth = computed(() => curveWidth - margin.left - margin.right);
const plotHeight = computed(() => curveHeight - margin.top - margin.bottom);
const selectedPoint = computed(() => props.layer.frequencyCurve[props.layer.selectedPointIndex] || props.layer.frequencyCurve[0]);

function frequencyToX(frequencyHz: number): number {
  const ratio = (Math.log(frequencyHz) - Math.log(minCurveFrequencyHz)) / curveLogSpan;
  return margin.left + (ratio * plotWidth.value);
}

function gainToY(gainDb: number): number {
  const ratio = (maxCurveGainDb - gainDb) / (maxCurveGainDb - minCurveGainDb);
  return margin.top + (ratio * plotHeight.value);
}

function buildCurvePath(): string {
  return props.layer.frequencyCurve.map((point, index) => {
    const command = index === 0 ? 'M' : 'L';
    return `${command} ${frequencyToX(point.frequency_hz).toFixed(2)} ${gainToY(point.gain_db).toFixed(2)}`;
  }).join(' ');
}

function buildCurveArea(): string {
  const points = props.layer.frequencyCurve;
  const startX = frequencyToX(points[0].frequency_hz).toFixed(2);
  const endX = frequencyToX(points[points.length - 1].frequency_hz).toFixed(2);
  const bottomY = (margin.top + plotHeight.value).toFixed(2);
  return `M ${startX} ${bottomY} ${buildCurvePath().slice(2)} L ${endX} ${bottomY} Z`;
}

const gainTicks = [maxCurveGainDb, 0, minCurveGainDb];
const freqTicks = [minCurveFrequencyHz, 27.5, 110.0, 440.0, 1760.0, maxCurveFrequencyHz];
</script>

<template>
  <div class="curve-panel">
    <div class="curve-toolbar">
      <button type="button" class="utility-btn" :disabled="layer.frequencyCurve.length >= 8" @click="emit('addPoint')">
        {{ t('curve.add_point') }}
      </button>
      <button type="button" class="utility-btn" :disabled="!canRemoveSelected" @click="emit('removeSelected')">
        {{ t('curve.remove_selected') }}
      </button>
      <button type="button" class="utility-btn" @click="emit('reset')">
        {{ t('curve.reset') }}
      </button>
    </div>
    <div class="curve-summary">{{ t('curve.drag_help') }}</div>
    <svg
      class="curve-svg"
      :id="`curveSvg${layerIndex}`"
      :viewBox="`0 0 ${curveWidth} ${curveHeight}`"
      :aria-label="t('curve.aria_label', { index: layerIndex + 1 })"
    >
      <rect :x="margin.left" :y="margin.top" :width="plotWidth" :height="plotHeight" fill="transparent"></rect>
      <g v-for="gainDb in gainTicks" :key="gainDb">
        <line
          :class="gainDb === 0 ? 'curve-zero-line' : 'curve-grid-line'"
          :x1="margin.left"
          :y1="gainToY(gainDb)"
          :x2="margin.left + plotWidth"
          :y2="gainToY(gainDb)"
        ></line>
        <text class="curve-axis-label" x="4" :y="gainToY(gainDb) + 4">{{ formatGainDb(gainDb) }}</text>
      </g>
      <g v-for="frequencyHz in freqTicks" :key="frequencyHz">
        <line
          class="curve-grid-line"
          :x1="frequencyToX(frequencyHz)"
          :y1="margin.top"
          :x2="frequencyToX(frequencyHz)"
          :y2="margin.top + plotHeight"
        ></line>
        <text class="curve-axis-label" :x="frequencyToX(frequencyHz)" :y="curveHeight - 6" text-anchor="middle">
          {{ frequencyHz >= 1000 ? `${(frequencyHz / 1000).toFixed(1)}k` : Math.round(frequencyHz) }}
        </text>
      </g>
      <path class="curve-fill" :d="buildCurveArea()"></path>
      <path class="curve-path" :d="buildCurvePath()"></path>
      <g v-for="(point, pointIndex) in layer.frequencyCurve" :key="`${point.frequency_hz}-${pointIndex}`">
        <circle
          class="curve-point-hit"
          :cx="frequencyToX(point.frequency_hz)"
          :cy="gainToY(point.gain_db)"
          r="7"
          @pointerdown="emit('startPointDrag', pointIndex, $event)"
          @click="emit('selectPoint', pointIndex)"
        ></circle>
        <circle
          class="curve-point"
          :class="{ selected: layer.selectedPointIndex === pointIndex }"
          :cx="frequencyToX(point.frequency_hz)"
          :cy="gainToY(point.gain_db)"
          :r="pointIndex === 0 || pointIndex === layer.frequencyCurve.length - 1 ? 3.4 : 3.0"
        ></circle>
      </g>
    </svg>
    <div class="curve-summary mt-2">
      {{ t('curve.points_summary', { count: layer.frequencyCurve.length, frequency: formatFrequency(selectedPoint.frequency_hz), gain: formatGainDb(selectedPoint.gain_db) }) }}
    </div>
  </div>
</template>
