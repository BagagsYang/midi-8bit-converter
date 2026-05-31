import type { FrequencyCurvePoint, SampleRate, WaveType, WorkspaceConfigV1, WorkspaceLayerConfig } from './types/api';
import type { LayerState } from './types/ui';

export const sampleRates: SampleRate[] = [44100, 48000, 96000];
export const waveTypeOptions: Array<[WaveType, string]> = [
  ['pulse', 'wave.pulse'],
  ['sine', 'wave.sine'],
  ['sawtooth', 'wave.sawtooth'],
  ['triangle', 'wave.triangle'],
  ['noise', 'wave.noise'],
];

export const standardWaveTypeOptions = waveTypeOptions.filter(([value]) => value !== 'noise');

const layerPresets: Array<Pick<LayerState, 'type' | 'duty' | 'volume'>> = [
  { type: 'pulse', duty: 0.5, volume: 1.0 },
  { type: 'sine', duty: 0.5, volume: 1.0 },
  { type: 'triangle', duty: 0.5, volume: 1.0 },
  { type: 'sawtooth', duty: 0.5, volume: 1.0 },
  { type: 'noise', duty: 0.5, volume: 1.0 },
  { type: 'pulse', duty: 0.25, volume: 1.0 },
  { type: 'sine', duty: 0.5, volume: 1.0 },
  { type: 'triangle', duty: 0.5, volume: 1.0 },
  { type: 'sawtooth', duty: 0.5, volume: 1.0 },
  { type: 'pulse', duty: 0.75, volume: 1.0 },
];

export const maxLayers = layerPresets.length;
export const minLayerOctaveShift = -2;
export const maxLayerOctaveShift = 2;
export const minCurveFrequencyHz = 8.175798915643707;
export const maxCurveFrequencyHz = 12543.853951415975;
export const minCurveGainDb = -36.0;
export const maxCurveGainDb = 12.0;
export const maxCurvePoints = 8;
export const curveWidth = 320;
export const curveHeight = 150;
export const curveMargin = { top: 14, right: 14, bottom: 24, left: 38 };

export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

export function createDefaultCurve(): FrequencyCurvePoint[] {
  return [
    { frequency_hz: minCurveFrequencyHz, gain_db: 0.0 },
    { frequency_hz: maxCurveFrequencyHz, gain_db: 0.0 },
  ];
}

export function createDefaultLayer(index: number): LayerState {
  const preset = layerPresets[index] || layerPresets[0];
  return {
    type: preset.type,
    duty: preset.duty,
    volume: preset.volume,
    curveEnabled: false,
    frequencyCurve: createDefaultCurve(),
    selectedPointIndex: 0,
    midiChannels: [],
    vibratoDepthCents: 0,
    vibratoRateHz: 5,
    octaveShift: 0,
  };
}

export function createDefaultLayers(): LayerState[] {
  return Array.from({ length: maxLayers }, (_, index) => createDefaultLayer(index));
}

export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 KB';
  }
  if (bytes < 1024 * 1024) {
    return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function isMidiFile(file: File): boolean {
  return file.type === 'audio/midi'
    || file.type === 'audio/x-midi'
    || /\.midi?$/i.test(file.name);
}

export function normaliseDecimalInput(value: string | number, min: number, max: number): number {
  const parsedValue = typeof value === 'number' ? value : parseFloat(value);
  const finiteValue = Number.isFinite(parsedValue) ? parsedValue : min;
  return Number(clamp(finiteValue, min, max).toFixed(2));
}

export function nextLayerOctaveShift(value: number, delta: number): number {
  const currentValue = Number.isFinite(Number(value)) ? Math.trunc(Number(value)) : 0;
  const deltaValue = Number.isFinite(Number(delta)) ? Math.trunc(Number(delta)) : 0;
  return clamp(currentValue + deltaValue, minLayerOctaveShift, maxLayerOctaveShift);
}

export function layerToConfig(layer: LayerState): WorkspaceLayerConfig {
  return {
    type: layer.type,
    duty: Number(layer.duty.toFixed(4)),
    volume: Number(layer.volume.toFixed(4)),
    curve_enabled: Boolean(layer.curveEnabled),
    frequency_curve: layer.frequencyCurve.map((point) => ({
      frequency_hz: point.frequency_hz,
      gain_db: Number(point.gain_db.toFixed(4)),
    })),
    pro: {
      midi_channels: [...new Set(layer.midiChannels)]
        .filter((channel) => Number.isInteger(channel) && channel >= 1 && channel <= 16)
        .sort((left, right) => left - right),
      vibrato_depth_cents: Number(clamp(layer.vibratoDepthCents, 0, 200).toFixed(2)),
      vibrato_rate_hz: Number(clamp(layer.vibratoRateHz, 0.1, 20).toFixed(2)),
      octave_shift: nextLayerOctaveShift(layer.octaveShift, 0),
    },
  };
}

export function currentWorkspaceConfig(sampleRate: SampleRate, layers: LayerState[], layerCount: number): WorkspaceConfigV1 {
  return {
    schema: 'octabit.workspace_config.v1',
    sample_rate: sampleRate,
    layers: layers.slice(0, layerCount).map(layerToConfig),
  };
}

export function layerFromConfig(configLayer: WorkspaceLayerConfig, index: number): LayerState {
  const fallback = createDefaultLayer(index);
  const validType = waveTypeOptions.some(([value]) => value === configLayer.type);
  return {
    ...fallback,
    type: validType ? configLayer.type : fallback.type,
    duty: Number.isFinite(Number(configLayer.duty)) ? Number(configLayer.duty) : fallback.duty,
    volume: Number.isFinite(Number(configLayer.volume)) ? Number(configLayer.volume) : fallback.volume,
    curveEnabled: Boolean(configLayer.curve_enabled),
    frequencyCurve: Array.isArray(configLayer.frequency_curve) && configLayer.frequency_curve.length
      ? configLayer.frequency_curve.map((point) => ({
        frequency_hz: Number(point.frequency_hz),
        gain_db: Number(point.gain_db),
      }))
      : createDefaultCurve(),
    selectedPointIndex: 0,
    midiChannels: Array.isArray(configLayer.pro?.midi_channels)
      ? [...new Set(configLayer.pro.midi_channels.map(Number))]
        .filter((channel) => Number.isInteger(channel) && channel >= 1 && channel <= 16)
        .sort((left, right) => left - right)
      : [],
    vibratoDepthCents: Number.isFinite(Number(configLayer.pro?.vibrato_depth_cents))
      ? clamp(Number(configLayer.pro?.vibrato_depth_cents), 0, 200)
      : 0,
    vibratoRateHz: Number.isFinite(Number(configLayer.pro?.vibrato_rate_hz))
      ? clamp(Number(configLayer.pro?.vibrato_rate_hz), 0.1, 20)
      : 5,
    octaveShift: Number.isFinite(Number(configLayer.pro?.octave_shift))
      ? nextLayerOctaveShift(Number(configLayer.pro?.octave_shift), 0)
      : 0,
  };
}

export function formatFrequency(value: number): string {
  return value >= 1000 ? `${(value / 1000).toFixed(2)} kHz` : `${value.toFixed(1)} Hz`;
}

export function formatGainDb(value: number): string {
  return `${value >= 0 ? '+' : ''}${value.toFixed(1)} dB`;
}

export function evaluateCurveGainDb(points: FrequencyCurvePoint[], frequencyHz: number): number {
  if (!points.length) return 0.0;
  if (frequencyHz <= points[0].frequency_hz) return points[0].gain_db;
  if (frequencyHz >= points[points.length - 1].frequency_hz) return points[points.length - 1].gain_db;
  for (let index = 0; index < points.length - 1; index += 1) {
    const leftPoint = points[index];
    const rightPoint = points[index + 1];
    if (frequencyHz >= leftPoint.frequency_hz && frequencyHz <= rightPoint.frequency_hz) {
      const leftLog = Math.log(leftPoint.frequency_hz);
      const rightLog = Math.log(rightPoint.frequency_hz);
      const frequencyLog = Math.log(frequencyHz);
      const ratio = (frequencyLog - leftLog) / (rightLog - leftLog);
      return leftPoint.gain_db + (ratio * (rightPoint.gain_db - leftPoint.gain_db));
    }
  }
  return points[points.length - 1].gain_db;
}
