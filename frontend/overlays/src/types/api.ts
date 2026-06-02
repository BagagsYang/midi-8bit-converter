export type WaveType = 'pulse' | 'sine' | 'sawtooth' | 'triangle' | 'noise';
export type SampleRate = 44100 | 48000 | 96000;
export type SynthesisJobStatus = 'queued' | 'rendering' | 'ready' | 'failed' | 'expired';

export interface ApiError {
  error?: {
    code?: string;
    message?: string;
  } | string;
  status?: SynthesisJobStatus;
}

export interface FrequencyCurvePoint {
  frequency_hz: number;
  gain_db: number;
}

export interface WorkspaceLayerConfig {
  type: WaveType;
  duty: number;
  volume: number;
  curve_enabled: boolean;
  frequency_curve: FrequencyCurvePoint[];
  pro?: WorkspaceLayerProConfig;
}

export interface WorkspaceLayerProConfig {
  midi_channels: number[];
  detune_cents: number;
  vibrato_depth_cents: number;
  vibrato_rate_hz: number;
  octave_shift: number;
}

export interface WorkspaceChannelBusConfig {
  channel: number;
  volume: number;
  mute: boolean;
  solo: boolean;
}

export interface WorkspaceConfigV1 {
  schema: 'octabit.workspace_config.v1';
  sample_rate: SampleRate;
  layers: WorkspaceLayerConfig[];
  channel_buses?: WorkspaceChannelBusConfig[];
  master_gain_db?: number;
  limiter_enabled?: boolean;
  normalise_enabled?: boolean;
}

export interface WorkspaceInfo {
  expires_at: number;
}

export interface WorkspaceLimits {
  max_queued_files: number;
  max_upload_bytes: number;
  max_converted_files: number;
}

export interface MIDIProfile {
  channels: number[];
  melodic_channels: number[];
  multi_channel: boolean;
}

export interface WorkspaceUpload {
  file_id: string;
  name: string;
  size: number;
  created_at: number;
  midi_profile: MIDIProfile;
}

export interface ConvertedFile {
  job_id: string;
  name: string;
  source_name: string;
  size: number;
  download_url: string;
  delete_url?: string;
  created_at: number;
  updated_at: number;
  expires_at: number;
}

export interface WorkspaceStateResponse {
  workspace: WorkspaceInfo;
  limits: WorkspaceLimits;
  config: WorkspaceConfigV1;
  uploads: WorkspaceUpload[];
  converted_files: ConvertedFile[];
}

export interface WorkspaceUploadResponse {
  upload: WorkspaceUpload;
}

export interface WorkspaceQueueResponse {
  uploads: WorkspaceUpload[];
}

export interface WorkspaceConfigResponse {
  config: WorkspaceConfigV1;
}

export interface SynthesisJobResponse {
  job_id: string;
  status: SynthesisJobStatus;
  source_name: string;
  created_at: number;
  updated_at: number;
  expires_at: number;
  download_name?: string;
  size_bytes?: number;
  download_url?: string;
  delete_url?: string;
}

export interface CreateSynthesisJobRequest {
  file_id: string;
  config: WorkspaceConfigV1;
}
