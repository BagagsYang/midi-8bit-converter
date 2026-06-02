import { describe, expect, it } from 'vitest';
import {
  currentWorkspaceConfig,
  createDefaultLayers,
  createDefaultLayer,
  layerFromConfig,
  layerToConfig,
  maxLayers,
  nextLayerOctaveShift,
} from './lib';
import type { WorkspaceLayerConfig } from './types/api';

describe('Pro octave shift layer config', () => {
  it('supports ten configured layers', () => {
    expect(maxLayers).toBe(10);
    expect(createDefaultLayers()).toHaveLength(10);
  });

  it('defaults to zero', () => {
    expect(createDefaultLayer(0).octaveShift).toBe(0);
    expect(createDefaultLayer(0).detuneCents).toBe(0);
  });

  it('serializes octave shift into pro config', () => {
    const layer = createDefaultLayer(0);
    layer.octaveShift = 2;
    layer.detuneCents = 12;

    expect(layerToConfig(layer).pro?.octave_shift).toBe(2);
    expect(layerToConfig(layer).pro?.detune_cents).toBe(12);
  });

  it('restores octave shift from pro config', () => {
    const configLayer: WorkspaceLayerConfig = {
      type: 'pulse',
      duty: 0.5,
      volume: 1,
      curve_enabled: false,
      frequency_curve: [],
      pro: {
        midi_channels: [],
        detune_cents: -8,
        vibrato_depth_cents: 0,
        vibrato_rate_hz: 5,
        octave_shift: -1,
      },
    };

    expect(layerFromConfig(configLayer, 0).octaveShift).toBe(-1);
    expect(layerFromConfig(configLayer, 0).detuneCents).toBe(-8);
  });

  it('serializes channel buses and output gain settings', () => {
    const config = currentWorkspaceConfig(48000, [createDefaultLayer(0)], 1, [
      { channel: 2, volume: 0.75, mute: true, solo: false },
    ], 3.5, true, false);

    expect(config.channel_buses).toEqual([
      { channel: 2, volume: 0.75, mute: true, solo: false },
    ]);
    expect(config.master_gain_db).toBe(3.5);
    expect(config.limiter_enabled).toBe(true);
    expect(config.normalise_enabled).toBe(false);
  });

  it('clamps octave shifts to the supported range', () => {
    expect(nextLayerOctaveShift(2, 1)).toBe(2);
    expect(nextLayerOctaveShift(-2, -1)).toBe(-2);
    expect(nextLayerOctaveShift(0, 1)).toBe(1);
    expect(nextLayerOctaveShift(0, -1)).toBe(-1);
  });
});
