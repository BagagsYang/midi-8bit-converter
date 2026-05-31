import type { FrequencyCurvePoint, MIDIProfile, WaveType } from './api';

export interface QueuedFile {
  fileId: string;
  name: string;
  size: number;
  midiProfile: MIDIProfile;
}

export interface ConvertedItem {
  jobId: string;
  name: string;
  sourceName: string;
  size: number;
  url: string;
  deleteUrl?: string;
}

export interface LayerState {
  type: WaveType;
  duty: number;
  volume: number;
  curveEnabled: boolean;
  frequencyCurve: FrequencyCurvePoint[];
  selectedPointIndex: number;
  midiChannels: number[];
  vibratoDepthCents: number;
  vibratoRateHz: number;
  octaveShift: number;
}

export type Translate = (key: string, params?: Record<string, string | number>) => string;
