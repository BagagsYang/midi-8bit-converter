import type { FrequencyCurvePoint, WaveType } from './api';

export interface QueuedFile {
  fileId: string;
  name: string;
  size: number;
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
}

export type Translate = (key: string, params?: Record<string, string | number>) => string;
