import { computed, ref, watchEffect } from 'vue';
import type { QueuedFile } from './types/ui';

const storageKey = 'octabitProMode';

export const proMode = ref(localStorage.getItem(storageKey) === 'true');

watchEffect(() => {
  localStorage.setItem(storageKey, String(proMode.value));
  document.documentElement.dataset.pro = proMode.value ? 'true' : 'false';
});

export function useProMode() {
  function toggleProMode() {
    proMode.value = !proMode.value;
  }

  return {
    proMode,
    toggleProMode,
  };
}

export function useProEligibility(queue: { value: QueuedFile[] }) {
  const hasQueuedFiles = computed(() => queue.value.length > 0);
  const allQueuedFilesMultiChannel = computed(() => (
    queue.value.length > 0 && queue.value.every((file) => file.midiProfile?.multi_channel)
  ));
  const availableChannels = computed(() => {
    const channels = new Set<number>();
    queue.value.forEach((file) => {
      const sourceChannels = file.midiProfile?.melodic_channels?.length
        ? file.midiProfile.melodic_channels
        : file.midiProfile?.channels || [];
      sourceChannels.forEach((channel) => channels.add(channel));
    });
    return Array.from(channels).sort((left, right) => left - right);
  });

  return {
    hasQueuedFiles,
    allQueuedFilesMultiChannel,
    availableChannels,
  };
}
