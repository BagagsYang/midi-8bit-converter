<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import {
  createSynthesisJob,
  deleteSynthesisJob,
  deleteWorkspaceUpload,
  getSynthesisJob,
  getWorkspace,
  responseErrorMessage,
  saveWorkspaceConfig,
  updateWorkspaceQueue,
  uploadWorkspaceFile,
} from './api/client';
import ConvertedFilesList from './components/ConvertedFilesList.vue';
import HeaderControls from './components/HeaderControls.vue';
import LayerEditor from './components/LayerEditor.vue';
import OutputControls from './components/OutputControls.vue';
import UploadQueue from './components/UploadQueue.vue';
import en from './i18n/en.json';
import fr from './i18n/fr.json';
import zhCn from './i18n/zh-CN.json';
import {
  clamp,
  createDefaultCurve,
  createDefaultLayer,
  createDefaultLayers,
  currentWorkspaceConfig,
  evaluateCurveGainDb,
  isMidiFile,
  layerFromConfig,
  maxCurveFrequencyHz,
  maxCurveGainDb,
  maxLayers,
  minCurveFrequencyHz,
  minCurveGainDb,
  sampleRates,
  waveTypeOptions,
} from './lib';
import type { SampleRate, SynthesisJobResponse, WaveType, WorkspaceConfigV1, WorkspaceUpload } from './types/api';
import type { ConvertedItem, LayerState, QueuedFile } from './types/ui';

type Locale = 'en' | 'fr' | 'zh-CN';
type ThemeChoice = 'system' | 'light' | 'dark';

const translationsByLocale: Record<Locale, Record<string, string>> = {
  en,
  fr,
  'zh-CN': zhCn,
};
const supportedLocales: Locale[] = ['en', 'fr', 'zh-CN'];
const defaultLocale: Locale = 'en';
const localeCookieName = 'web_locale';
const themeStorageKey = 'octabitTheme';
const workspaceConfigSaveDelayMs = 400;

const locale = ref<Locale>(initialLocale());
const themeChoice = ref<ThemeChoice>(initialThemeChoice());
const activeTheme = ref<'light' | 'dark'>('dark');
const queue = ref<QueuedFile[]>([]);
const convertedFiles = ref<ConvertedItem[]>([]);
const keepQueue = ref(localStorage.getItem('keepQueueAfterSynth') === 'true');
const sampleRate = ref<SampleRate>(48000);
const layerCount = ref(1);
const layers = ref<LayerState[]>(createDefaultLayers());
const isRestoringWorkspace = ref(false);
const isProcessing = ref(false);
const processingStatus = ref('');
const previewAudio = new Audio();
previewAudio.volume = 0.5;

let workspaceConfigSaveTimer: number | null = null;
let dragState: { layerIndex: number; pointIndex: number } | null = null;

const translations = computed(() => translationsByLocale[locale.value] || translationsByLocale[defaultLocale]);

function t(key: string, params: Record<string, string | number> = {}): string {
  const template = Object.prototype.hasOwnProperty.call(translations.value, key) ? translations.value[key] : key;
  return template.replace(/\{(\w+)\}/g, (match, token) => (
    Object.prototype.hasOwnProperty.call(params, token) ? String(params[token]) : match
  ));
}

function initialLocale(): Locale {
  const urlLocale = new URLSearchParams(window.location.search).get('lang');
  const cookieLocale = document.cookie
    .split('; ')
    .find((part) => part.startsWith(`${localeCookieName}=`))
    ?.split('=')[1];
  return normaliseLocale(urlLocale || (cookieLocale ? decodeURIComponent(cookieLocale) : null));
}

function normaliseLocale(value: string | null): Locale {
  return supportedLocales.includes(value as Locale) ? value as Locale : defaultLocale;
}

function updateLocale(value: string) {
  const nextLocale = normaliseLocale(value);
  locale.value = nextLocale;
  document.documentElement.lang = nextLocale;
  document.title = t('meta.site_title');
  document.cookie = `${localeCookieName}=${encodeURIComponent(nextLocale)}; path=/; max-age=${60 * 60 * 24 * 365}; SameSite=Lax`;
  const url = new URL(window.location.href);
  if (nextLocale === defaultLocale) {
    url.searchParams.delete('lang');
  } else {
    url.searchParams.set('lang', nextLocale);
  }
  window.history.replaceState({}, '', url.toString());
}

function systemTheme(): 'light' | 'dark' {
  if (window.matchMedia?.('(prefers-color-scheme: light)').matches) {
    return 'light';
  }
  return 'dark';
}

function initialThemeChoice(): ThemeChoice {
  const stored = localStorage.getItem(themeStorageKey);
  return stored === 'light' || stored === 'dark' ? stored : 'system';
}

function applyThemeChoice(choice: ThemeChoice) {
  themeChoice.value = choice;
  if (choice === 'system') {
    localStorage.removeItem(themeStorageKey);
    activeTheme.value = systemTheme();
  } else {
    localStorage.setItem(themeStorageKey, choice);
    activeTheme.value = choice;
  }
  document.documentElement.classList.add('theme-change-instant');
  document.documentElement.setAttribute('data-bs-theme', activeTheme.value);
  void document.documentElement.offsetHeight;
  document.documentElement.classList.remove('theme-change-instant');
}

function syncSystemTheme() {
  if (themeChoice.value === 'system') {
    applyThemeChoice('system');
  }
}

function uploadRecordFromApi(upload: WorkspaceUpload): QueuedFile {
  return {
    fileId: upload.file_id,
    name: upload.name,
    size: upload.size,
  };
}

function convertedRecordFromApi(file: {
  job_id: string;
  name: string;
  source_name: string;
  size: number;
  download_url: string;
  delete_url?: string;
}): ConvertedItem {
  return {
    jobId: file.job_id,
    name: file.name,
    sourceName: file.source_name,
    size: file.size,
    url: new URL(file.download_url, window.location.origin).toString(),
    deleteUrl: file.delete_url,
  };
}

function applyWorkspaceConfig(config: WorkspaceConfigV1) {
  if (!config || !Array.isArray(config.layers)) {
    return;
  }
  if (sampleRates.includes(Number(config.sample_rate) as SampleRate)) {
    sampleRate.value = Number(config.sample_rate) as SampleRate;
  }
  layerCount.value = Math.min(Math.max(config.layers.length, 1), maxLayers);
  const nextLayers = createDefaultLayers();
  config.layers.slice(0, layerCount.value).forEach((configLayer, index) => {
    nextLayers[index] = layerFromConfig(configLayer, index);
  });
  layers.value = nextLayers;
}

async function restoreWorkspace() {
  try {
    const payload = await getWorkspace();
    isRestoringWorkspace.value = true;
    queue.value = Array.isArray(payload.uploads) ? payload.uploads.map(uploadRecordFromApi) : [];
    convertedFiles.value = Array.isArray(payload.converted_files) ? payload.converted_files.map(convertedRecordFromApi) : [];
    applyWorkspaceConfig(payload.config);
  } catch (error) {
    console.warn('Failed to restore workspace.', error);
  } finally {
    isRestoringWorkspace.value = false;
  }
}

function configPayload(): WorkspaceConfigV1 {
  return currentWorkspaceConfig(sampleRate.value, layers.value, layerCount.value);
}

function scheduleWorkspaceConfigSave() {
  if (isRestoringWorkspace.value) {
    return;
  }
  if (workspaceConfigSaveTimer !== null) {
    window.clearTimeout(workspaceConfigSaveTimer);
  }
  workspaceConfigSaveTimer = window.setTimeout(async () => {
    workspaceConfigSaveTimer = null;
    try {
      await saveWorkspaceConfig(configPayload());
    } catch (error) {
      console.warn('Failed to save workspace config.', error);
    }
  }, workspaceConfigSaveDelayMs);
}

async function addFiles(files: FileList | File[]) {
  const uploadedFiles: QueuedFile[] = [];
  for (const file of Array.from(files)) {
    if (!isMidiFile(file)) {
      continue;
    }
    try {
      const payload = await uploadWorkspaceFile(file);
      uploadedFiles.push(uploadRecordFromApi(payload.upload));
    } catch (error) {
      alert(t('alerts.upload_error', {
        filename: file.name,
        error: error instanceof Error ? error.message : t('alerts.processing_unknown', { filename: file.name }),
      }));
    }
  }
  queue.value = [...queue.value, ...uploadedFiles];
}

async function removeFromQueue(index: number) {
  const nextQueue = [...queue.value];
  const [file] = nextQueue.splice(index, 1);
  queue.value = nextQueue;
  if (!file) return;
  try {
    await deleteWorkspaceUpload(file.fileId);
  } catch (error) {
    console.warn('Failed to delete workspace upload.', error);
  }
}

async function clearQueue() {
  const filesToClear = [...queue.value];
  queue.value = [];
  await Promise.all(filesToClear.map((file) => (
    deleteWorkspaceUpload(file.fileId).catch((error) => {
      console.warn('Failed to delete workspace upload.', error);
    })
  )));
}

async function reorderQueue(fromIndex: number, toIndex: number) {
  const nextQueue = [...queue.value];
  const [draggedItem] = nextQueue.splice(fromIndex, 1);
  if (!draggedItem) return;
  nextQueue.splice(toIndex, 0, draggedItem);
  queue.value = nextQueue;
  try {
    await updateWorkspaceQueue(queue.value.map((file) => file.fileId));
  } catch (error) {
    console.warn('Failed to save queue order.', error);
  }
}

function setKeepQueue(value: boolean) {
  keepQueue.value = value;
  localStorage.setItem('keepQueueAfterSynth', String(value));
}

function setSampleRate(value: SampleRate) {
  sampleRate.value = value;
  scheduleWorkspaceConfigSave();
}

function activeLayerTypes(excludedLayerIndex: number | null = null): Set<WaveType> {
  return new Set(layers.value.slice(0, layerCount.value)
    .filter((_layer, index) => index !== excludedLayerIndex)
    .map((layer) => layer.type));
}

function firstUnusedWaveType(): WaveType | null {
  const usedTypes = activeLayerTypes();
  const option = waveTypeOptions.find(([value]) => !usedTypes.has(value));
  return option ? option[0] : null;
}

function updateLayerType(layerIndex: number, value: WaveType) {
  const validWaveType = waveTypeOptions.some(([optionValue]) => optionValue === value);
  if (!validWaveType || activeLayerTypes(layerIndex).has(value)) {
    return;
  }
  layers.value[layerIndex].type = value;
  scheduleWorkspaceConfigSave();
}

function updateLayerDuty(layerIndex: number, value: number) {
  layers.value[layerIndex].duty = value;
  scheduleWorkspaceConfigSave();
}

function updateLayerVolume(layerIndex: number, value: number) {
  layers.value[layerIndex].volume = value;
  scheduleWorkspaceConfigSave();
}

function toggleCurve(layerIndex: number, enabled: boolean) {
  const layer = layers.value[layerIndex];
  layer.curveEnabled = enabled;
  if (!layer.frequencyCurve.length) {
    layer.frequencyCurve = createDefaultCurve();
    layer.selectedPointIndex = 0;
  }
  scheduleWorkspaceConfigSave();
}

function addCurvePoint(layerIndex: number) {
  const layer = layers.value[layerIndex];
  if (layer.frequencyCurve.length >= 8) return;
  let widestGapIndex = 0;
  let widestGap = -1;
  for (let index = 0; index < layer.frequencyCurve.length - 1; index += 1) {
    const gap = Math.log(layer.frequencyCurve[index + 1].frequency_hz) - Math.log(layer.frequencyCurve[index].frequency_hz);
    if (gap > widestGap) {
      widestGap = gap;
      widestGapIndex = index;
    }
  }
  const leftPoint = layer.frequencyCurve[widestGapIndex];
  const rightPoint = layer.frequencyCurve[widestGapIndex + 1];
  const newFrequency = Math.sqrt(leftPoint.frequency_hz * rightPoint.frequency_hz);
  const newGain = evaluateCurveGainDb(layer.frequencyCurve, newFrequency);
  layer.frequencyCurve.splice(widestGapIndex + 1, 0, {
    frequency_hz: newFrequency,
    gain_db: newGain,
  });
  layer.selectedPointIndex = widestGapIndex + 1;
  scheduleWorkspaceConfigSave();
}

function removeSelectedPoint(layerIndex: number) {
  const layer = layers.value[layerIndex];
  if (layer.selectedPointIndex <= 0 || layer.selectedPointIndex >= layer.frequencyCurve.length - 1) {
    return;
  }
  layer.frequencyCurve.splice(layer.selectedPointIndex, 1);
  layer.selectedPointIndex = Math.max(0, layer.selectedPointIndex - 1);
  scheduleWorkspaceConfigSave();
}

function resetCurve(layerIndex: number) {
  layers.value[layerIndex].frequencyCurve = createDefaultCurve();
  layers.value[layerIndex].selectedPointIndex = 0;
  scheduleWorkspaceConfigSave();
}

function selectCurvePoint(layerIndex: number, pointIndex: number) {
  layers.value[layerIndex].selectedPointIndex = pointIndex;
}

function addLayer() {
  if (layerCount.value >= maxLayers) return;
  const unusedType = firstUnusedWaveType();
  if (!unusedType) return;
  layers.value[layerCount.value] = createDefaultLayer(layerCount.value);
  layers.value[layerCount.value].type = unusedType;
  layerCount.value += 1;
  scheduleWorkspaceConfigSave();
}

function removeLayer() {
  if (layerCount.value <= 1) return;
  layers.value[layerCount.value - 1] = createDefaultLayer(layerCount.value - 1);
  layerCount.value -= 1;
  scheduleWorkspaceConfigSave();
}

function resetLayers() {
  layers.value = createDefaultLayers();
  layerCount.value = 1;
  scheduleWorkspaceConfigSave();
}

function playPreview(layerIndex: number) {
  const layer = layers.value[layerIndex];
  let src = `${layer.type}.wav`;
  if (layer.type === 'pulse') {
    src = `pulse_${layer.duty < 0.18 ? '10' : layer.duty < 0.38 ? '25' : '50'}.wav`;
  }
  previewAudio.src = `/static/previews/${src}`;
  previewAudio.play().catch((error) => console.error('Preview failed:', error));
}

function startCurvePointDrag(layerIndex: number, pointIndex: number, event: PointerEvent) {
  event.preventDefault();
  layers.value[layerIndex].selectedPointIndex = pointIndex;
  dragState = { layerIndex, pointIndex };
}

function xToFrequency(x: number): number {
  const curveWidth = 320;
  const marginLeft = 38;
  const marginRight = 14;
  const plotWidth = curveWidth - marginLeft - marginRight;
  const ratio = clamp((x - marginLeft) / plotWidth, 0, 1);
  return minCurveFrequencyHz * ((maxCurveFrequencyHz / minCurveFrequencyHz) ** ratio);
}

function yToGain(y: number): number {
  const curveHeight = 150;
  const marginTop = 14;
  const marginBottom = 24;
  const plotHeight = curveHeight - marginTop - marginBottom;
  const ratio = clamp((y - marginTop) / plotHeight, 0, 1);
  return maxCurveGainDb - (ratio * (maxCurveGainDb - minCurveGainDb));
}

function onPointerMove(event: PointerEvent) {
  if (!dragState) return;
  const { layerIndex, pointIndex } = dragState;
  const svg = document.getElementById(`curveSvg${layerIndex}`) || document.querySelectorAll('.curve-svg')[layerIndex];
  if (!svg) return;
  const rect = svg.getBoundingClientRect();
  const localX = ((event.clientX - rect.left) / rect.width) * 320;
  const localY = ((event.clientY - rect.top) / rect.height) * 150;
  const layer = layers.value[layerIndex];
  const points = layer.frequencyCurve;
  const point = points[pointIndex];
  point.gain_db = Number(clamp(yToGain(localY), minCurveGainDb, maxCurveGainDb).toFixed(4));
  if (pointIndex === 0) {
    point.frequency_hz = minCurveFrequencyHz;
  } else if (pointIndex === points.length - 1) {
    point.frequency_hz = maxCurveFrequencyHz;
  } else {
    const minFrequency = points[pointIndex - 1].frequency_hz * 1.0001;
    const maxFrequency = points[pointIndex + 1].frequency_hz / 1.0001;
    point.frequency_hz = clamp(xToFrequency(localX), minFrequency, maxFrequency);
  }
  scheduleWorkspaceConfigSave();
}

function onPointerUp() {
  dragState = null;
}

function downloadConvertedFile(index: number) {
  const convertedFile = convertedFiles.value[index];
  if (!convertedFile) return;
  const anchor = document.createElement('a');
  anchor.href = convertedFile.url;
  anchor.download = convertedFile.name;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
}

async function clearConvertedFiles() {
  if (!window.confirm(t('converted.clear_confirm'))) {
    return;
  }
  const filesToClear = [...convertedFiles.value];
  convertedFiles.value = [];
  await Promise.all(filesToClear.map((file) => (
    deleteSynthesisJob(file.deleteUrl || file.jobId).catch((error) => {
      console.warn('Failed to delete converted server file.', error);
    })
  )));
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

async function waitForSynthesisJob(jobId: string, file: QueuedFile, index: number, total: number): Promise<SynthesisJobResponse> {
  while (true) {
    try {
      const payload = await getSynthesisJob(jobId);
      if (payload.status === 'ready') {
        processingStatus.value = t('status.file_ready', { current: index + 1, total, filename: file.name });
        return payload;
      }
      if (payload.status === 'failed' || payload.status === 'expired') {
        throw new Error(payload.status);
      }
      processingStatus.value = t('status.rendering_file', { current: index + 1, total, filename: file.name });
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error(String(error));
    }
    await sleep(1000);
  }
}

function addConvertedServerFile(job: SynthesisJobResponse, sourceName: string) {
  const jobId = job.job_id;
  convertedFiles.value = convertedFiles.value.filter((file) => file.jobId !== jobId);
  convertedFiles.value.unshift({
    jobId,
    name: job.download_name || `${sourceName.replace(/\.[^.]+$/, '') || 'output'}.wav`,
    sourceName,
    size: job.size_bytes || 0,
    url: new URL(job.download_url || `/api/synthesis-jobs/${jobId}/download`, window.location.origin).toString(),
    deleteUrl: job.delete_url,
  });
}

async function processQueue() {
  isProcessing.value = true;
  const filesToProcess = [...queue.value];
  const failedFiles: QueuedFile[] = [];
  const config = configPayload();

  for (let index = 0; index < filesToProcess.length; index += 1) {
    const file = filesToProcess[index];
    processingStatus.value = t('status.processing_file', {
      current: index + 1,
      total: filesToProcess.length,
      filename: file.name,
    });
    try {
      const job = await createSynthesisJob({ file_id: file.fileId, config });
      const readyJob = job.status === 'ready'
        ? job
        : await waitForSynthesisJob(job.job_id, file, index, filesToProcess.length);
      addConvertedServerFile(readyJob, file.name);
      processingStatus.value = t('status.downloading_file', {
        current: index + 1,
        total: filesToProcess.length,
        filename: file.name,
      });
      downloadConvertedFile(0);
    } catch (error) {
      failedFiles.push(file);
      alert(t('alerts.processing_error', {
        filename: file.name,
        error: error instanceof Error ? error.message : responseErrorMessage(error, t('alerts.processing_unknown', { filename: file.name })),
      }));
    }
  }

  isProcessing.value = false;
  processingStatus.value = t('status.generating_audio');
  if (!keepQueue.value) {
    const failedFileIds = new Set(failedFiles.map((file) => file.fileId));
    const processedFiles = filesToProcess.filter((file) => !failedFileIds.has(file.fileId));
    await Promise.all(processedFiles.map((file) => (
      deleteWorkspaceUpload(file.fileId).catch((error) => {
        console.warn('Failed to delete processed workspace upload.', error);
      })
    )));
    queue.value = failedFiles;
  }
}

onMounted(() => {
  document.documentElement.lang = locale.value;
  document.title = t('meta.site_title');
  processingStatus.value = t('status.generating_audio');
  applyThemeChoice(themeChoice.value);
  restoreWorkspace();
  window.addEventListener('pointermove', onPointerMove);
  window.addEventListener('pointerup', onPointerUp);
  window.matchMedia?.('(prefers-color-scheme: light)').addEventListener('change', syncSystemTheme);
  window.matchMedia?.('(prefers-color-scheme: dark)').addEventListener('change', syncSystemTheme);
});

onUnmounted(() => {
  if (workspaceConfigSaveTimer !== null) {
    window.clearTimeout(workspaceConfigSaveTimer);
  }
  window.removeEventListener('pointermove', onPointerMove);
  window.removeEventListener('pointerup', onPointerUp);
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
            :files="queue"
            :keep-queue="keepQueue"
            @add-files="addFiles"
            @remove-file="removeFromQueue"
            @clear-queue="clearQueue"
            @reorder="reorderQueue"
            @update:keep-queue="setKeepQueue"
          />
          <OutputControls
            :t="t"
            :sample-rate="sampleRate"
            :queue-count="queue.length"
            :is-processing="isProcessing"
            :processing-status="processingStatus"
            @update:sample-rate="setSampleRate"
            @process="processQueue"
          />
          <ConvertedFilesList
            :t="t"
            :files="convertedFiles"
            @download="downloadConvertedFile"
            @clear="clearConvertedFiles"
          />
        </aside>

        <main class="parameter-column" :aria-label="t('parameters.title')">
          <LayerEditor
            :t="t"
            :layers="layers"
            :layer-count="layerCount"
            @update-layer-type="updateLayerType"
            @update-layer-duty="updateLayerDuty"
            @update-layer-volume="updateLayerVolume"
            @toggle-curve="toggleCurve"
            @add-curve-point="addCurvePoint"
            @remove-selected-point="removeSelectedPoint"
            @reset-curve="resetCurve"
            @select-curve-point="selectCurvePoint"
            @start-curve-point-drag="startCurvePointDrag"
            @add-layer="addLayer"
            @remove-layer="removeLayer"
            @reset-layers="resetLayers"
            @play-preview="playPreview"
          />
        </main>
      </div>
    </div>
  </div>
</template>
