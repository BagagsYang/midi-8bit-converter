import { describe, expect, it } from 'vitest';
import { ref } from 'vue';
import { useProEligibility } from './pro';
import type { QueuedFile } from './types/ui';

describe('Pro channel eligibility', () => {
  it('uses only note-present melodic channels when building available channels', () => {
    const queue = ref<QueuedFile[]>([
      {
        fileId: 'file-1',
        name: 'song.mid',
        size: 1,
        midiProfile: {
          channels: [1, 2, 8],
          melodic_channels: [1, 8],
          multi_channel: true,
        },
      },
    ]);

    const { availableChannels } = useProEligibility(queue);

    expect(availableChannels.value).toEqual([1, 8]);
  });
});
