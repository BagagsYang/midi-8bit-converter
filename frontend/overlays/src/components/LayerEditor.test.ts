import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it } from 'vitest';
import { createDefaultLayer } from '../lib';
import { proMode } from '../pro';
import type { QueuedFile, Translate } from '../types/ui';
import LayerEditor from './LayerEditor.vue';

const labels: Record<string, string> = {
  'parameters.title': 'Parameter Control',
  'parameters.reset': 'Reset',
  'parameters.readout': '4 MAX',
  'layer.title': 'Layer 1',
  'layer.play_preview': 'Play Preview',
  'layer.waveform_type': 'Waveform Type',
  'layer.pulse_width': 'Pulse Width',
  'layer.base_volume': 'Base Volume',
  'layer.enable_curve': 'frequency-gain curve',
  'pro.channel': 'Channel 1',
  'pro.midi_channels': 'MIDI Channels',
  'pro.octave_shift': 'Octave Shift',
  'pro.octave_down': '-1 Oct',
  'pro.octave_up': '+1 Oct',
  'pro.octave_down_aria': 'Shift layer down one octave',
  'pro.octave_up_aria': 'Shift layer up one octave',
  'pro.vibrato_depth': 'Vibrato Depth',
  'pro.vibrato_rate': 'Vibrato Rate',
  'pro.requires_queue': 'Queue a MIDI file with at least two melodic channels.',
  'pro.requires_multi_channel': 'Pro note-shaping requires every queued MIDI file to use at least two melodic channels.',
  'pro.multi_channel_ready': 'Channel routing, octave shift, noise, and vibrato are available.',
  'wave.pulse': 'Square/Pulse',
  'wave.sine': 'Sine',
  'wave.sawtooth': 'Sawtooth',
  'wave.triangle': 'Triangle',
  'wave.noise': 'Noise',
  'layers.add': '+ Add Layer',
  'layers.remove': '- Remove Layer',
};

const t: Translate = (key) => labels[key] || key;

const multiChannelFile: QueuedFile = {
  fileId: 'file-1',
  name: 'multi.mid',
  size: 1,
  midiProfile: {
    channels: [1, 2],
    melodic_channels: [1, 2],
    multi_channel: true,
  },
};

function mountEditor(octaveShift = 0, queuedFiles: QueuedFile[] = [multiChannelFile]) {
  const layer = createDefaultLayer(0);
  layer.octaveShift = octaveShift;
  return mount(LayerEditor, {
    props: {
      t,
      layers: [layer],
      layerCount: 1,
      queuedFiles,
    },
  });
}

describe('LayerEditor Pro octave controls', () => {
  beforeEach(() => {
    proMode.value = true;
  });

  it('renders two octave buttons with a signed readout and emits deltas', async () => {
    const wrapper = mountEditor();
    const octaveButtons = wrapper.findAll('button')
      .filter((button) => ['-1 Oct', '+1 Oct'].includes(button.text()));

    expect(octaveButtons).toHaveLength(2);
    expect(wrapper.text()).toContain('Octave Shift');
    expect(wrapper.text()).toContain('0');

    await octaveButtons[0].trigger('click');
    await octaveButtons[1].trigger('click');

    expect(wrapper.emitted('updateLayerOctaveShift')).toEqual([
      [0, -1],
      [0, 1],
    ]);
  });

  it('disables only the bounded octave direction at range edges', () => {
    const minWrapper = mountEditor(-2);
    const minButtons = minWrapper.findAll('button')
      .filter((button) => ['-1 Oct', '+1 Oct'].includes(button.text()));
    expect(minButtons[0].attributes('disabled')).toBeDefined();
    expect(minButtons[1].attributes('disabled')).toBeUndefined();
    expect(minWrapper.text()).toContain('-2');

    const maxWrapper = mountEditor(2);
    const maxButtons = maxWrapper.findAll('button')
      .filter((button) => ['-1 Oct', '+1 Oct'].includes(button.text()));
    expect(maxButtons[0].attributes('disabled')).toBeUndefined();
    expect(maxButtons[1].attributes('disabled')).toBeDefined();
    expect(maxWrapper.text()).toContain('+2');
  });

  it('disables octave controls when Pro eligibility is not met', () => {
    const wrapper = mountEditor(0, []);
    const octaveButtons = wrapper.findAll('button')
      .filter((button) => ['-1 Oct', '+1 Oct'].includes(button.text()));

    expect(octaveButtons).toHaveLength(2);
    expect(octaveButtons.every((button) => button.attributes('disabled') !== undefined)).toBe(true);
  });
});
