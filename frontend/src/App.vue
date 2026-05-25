<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import ConvertedFilesList from './components/ConvertedFilesList.vue';
import HeaderControls from './components/HeaderControls.vue';
import LayerEditor from './components/LayerEditor.vue';
import OutputControls from './components/OutputControls.vue';
import UploadQueue from './components/UploadQueue.vue';
import { useLocale } from './composables/useLocale';
import { useTheme } from './composables/useTheme';
import { useWorkspace } from './composables/useWorkspace';

const { locale, t, supportedLocales, updateLocale } = useLocale();
const { themeChoice, activeTheme, applyThemeChoice, syncSystemTheme } = useTheme();
const workspace = useWorkspace(t);

onMounted(() => {
  document.documentElement.lang = locale.value;
  document.title = t('meta.site_title');
  workspace.processingStatus.value = t('status.generating_audio');
  applyThemeChoice(themeChoice.value);
  workspace.restoreWorkspace();
  window.addEventListener('pointermove', workspace.onPointerMove);
  window.addEventListener('pointerup', workspace.onPointerUp);
  window.matchMedia?.('(prefers-color-scheme: light)').addEventListener('change', syncSystemTheme);
  window.matchMedia?.('(prefers-color-scheme: dark)').addEventListener('change', syncSystemTheme);
});

onUnmounted(() => {
  workspace.cleanup();
  window.removeEventListener('pointermove', workspace.onPointerMove);
  window.removeEventListener('pointerup', workspace.onPointerUp);
  window.matchMedia?.('(prefers-color-scheme: light)').removeEventListener('change', syncSystemTheme);
  window.matchMedia?.('(prefers-color-scheme: dark)').removeEventListener('change', syncSystemTheme);
});
</script>

<template>
  <div class="container-main">
    <div class="control-shell">
      <HeaderControls
        :t="t"
        :locale="locale"
        :supported-locales="supportedLocales"
        :theme-choice="themeChoice"
        :active-theme="activeTheme"
        @update:locale="updateLocale"
        @update:theme-choice="applyThemeChoice"
      />

      <div class="workspace-grid">
        <aside class="file-column" :aria-label="t('files.operations')">
          <UploadQueue
            :t="t"
            :files="workspace.queue.value"
            :keep-queue="workspace.keepQueue.value"
            @add-files="workspace.addFiles"
            @remove-file="workspace.removeFromQueue"
            @clear-queue="workspace.clearQueue"
            @reorder="workspace.reorderQueue"
            @update:keep-queue="workspace.setKeepQueue"
          />
          <OutputControls
            :t="t"
            :sample-rate="workspace.sampleRate.value"
            :queue-count="workspace.queue.value.length"
            :is-processing="workspace.isProcessing.value"
            :processing-status="workspace.processingStatus.value"
            @update:sample-rate="workspace.setSampleRate"
            @process="workspace.processQueue"
          />
          <ConvertedFilesList
            :t="t"
            :files="workspace.convertedFiles.value"
            @download="workspace.downloadConvertedFile"
            @clear="workspace.clearConvertedFiles"
          />
        </aside>

        <main class="parameter-column" :aria-label="t('parameters.title')">
          <LayerEditor
            :t="t"
            :layers="workspace.layers.value"
            :layer-count="workspace.layerCount.value"
            @update-layer-type="workspace.updateLayerType"
            @update-layer-duty="workspace.updateLayerDuty"
            @update-layer-volume="workspace.updateLayerVolume"
            @toggle-curve="workspace.toggleCurve"
            @add-curve-point="workspace.addCurvePoint"
            @remove-selected-point="workspace.removeSelectedPoint"
            @reset-curve="workspace.resetCurve"
            @select-curve-point="workspace.selectCurvePoint"
            @start-curve-point-drag="workspace.startCurvePointDrag"
            @add-layer="workspace.addLayer"
            @remove-layer="workspace.removeLayer"
            @reset-layers="workspace.resetLayers"
            @play-preview="workspace.playPreview"
          />
        </main>
      </div>
    </div>
  </div>
</template>
