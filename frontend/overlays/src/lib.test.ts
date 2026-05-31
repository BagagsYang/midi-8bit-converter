import { describe, expect, it } from 'vitest';
import {
  createDefaultLayer,
  layerFromConfig,
  layerToConfig,
  nextLayerOctaveShift,
} from './lib';
import type { WorkspaceLayerConfig } from './types/api';

describe('Pro octave shift layer config', () => {
  it('defaults to zero', () => {
    expect(createDefaultLayer(0).octaveShift).toBe(0);
  });

  it('serializes octave shift into pro config', () => {
    const layer = createDefaultLayer(0);
    layer.octaveShift = 2;

    expect(layerToConfig(layer).pro?.octave_shift).toBe(2);
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
        vibrato_depth_cents: 0,
        vibrato_rate_hz: 5,
        octave_shift: -1,
      },
    };

    expect(layerFromConfig(configLayer, 0).octaveShift).toBe(-1);
  });

  it('clamps octave shifts to the supported range', () => {
    expect(nextLayerOctaveShift(2, 1)).toBe(2);
    expect(nextLayerOctaveShift(-2, -1)).toBe(-2);
    expect(nextLayerOctaveShift(0, 1)).toBe(1);
    expect(nextLayerOctaveShift(0, -1)).toBe(-1);
  });
});
