import { ref } from 'vue';
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
} from '../api/client';
import {
  clamp,
  createDefaultCurve,
  createDefaultLayer,
  createDefaultLayers,
  currentWorkspaceConfig,
  curveHeight,
  curveMargin,
  curveWidth,
  evaluateCurveGainDb,
  isMidiFile,
  layerFromConfig,
  maxCurveFrequencyHz,
  maxCurveGainDb,
  maxLayers,
  minCurveFrequencyHz,
  minCurveGainDb,
  nextLayerOctaveShift,
  sampleRates,
  waveTypeOptions,
} from '../lib';
import { proMode } from '../pro';
import type { SampleRate, SynthesisJobResponse, WaveType, WorkspaceConfigV1, WorkspaceUpload } from '../types/api';
import type { ConvertedItem, LayerState, QueuedFile } from '../types/ui';

const workspaceConfigSaveDelayMs = 400;

type Translate = (key: string, params?: Record<string, string | number>) => string;

function uploadRecordFromApi(upload: WorkspaceUpload): QueuedFile {
  return {
    fileId: upload.file_id,
    name: upload.name,
    size: upload.size,
    midiProfile: upload.midi_profile || { channels: [], melodic_channels: [], multi_channel: false },
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

export function useWorkspace(t: Translate) {
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
    if (!validWaveType || (!proMode.value && activeLayerTypes(layerIndex).has(value))) {
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

  function updateLayerMidiChannels(layerIndex: number, channels: number[]) {
    layers.value[layerIndex].midiChannels = [...new Set(channels)]
      .filter((channel) => Number.isInteger(channel) && channel >= 1 && channel <= 16)
      .sort((left, right) => left - right);
    scheduleWorkspaceConfigSave();
  }

  function updateLayerVibratoDepth(layerIndex: number, value: number) {
    layers.value[layerIndex].vibratoDepthCents = clamp(value, 0, 200);
    scheduleWorkspaceConfigSave();
  }

  function updateLayerVibratoRate(layerIndex: number, value: number) {
    layers.value[layerIndex].vibratoRateHz = clamp(value, 0.1, 20);
    scheduleWorkspaceConfigSave();
  }

  function updateLayerOctaveShift(layerIndex: number, delta: number) {
    layers.value[layerIndex].octaveShift = nextLayerOctaveShift(layers.value[layerIndex].octaveShift, delta);
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
    if (!unusedType && !proMode.value) return;
    layers.value[layerCount.value] = createDefaultLayer(layerCount.value);
    if (unusedType) {
      layers.value[layerCount.value].type = unusedType;
    }
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
    if (layer.type === 'noise') {
      return;
    }
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
    const marginLeft = curveMargin.left;
    const marginRight = curveMargin.right;
    const plotWidth = curveWidth - marginLeft - marginRight;
    const ratio = clamp((x - marginLeft) / plotWidth, 0, 1);
    return minCurveFrequencyHz * ((maxCurveFrequencyHz / minCurveFrequencyHz) ** ratio);
  }

  function yToGain(y: number): number {
    const marginTop = curveMargin.top;
    const marginBottom = curveMargin.bottom;
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
    const localX = ((event.clientX - rect.left) / rect.width) * curveWidth;
    const localY = ((event.clientY - rect.top) / rect.height) * curveHeight;
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

  function cleanup() {
    if (workspaceConfigSaveTimer !== null) {
      window.clearTimeout(workspaceConfigSaveTimer);
    }
  }

  return {
    queue,
    convertedFiles,
    keepQueue,
    sampleRate,
    layerCount,
    layers,
    isRestoringWorkspace,
    isProcessing,
    processingStatus,
    restoreWorkspace,
    addFiles,
    removeFromQueue,
    clearQueue,
    reorderQueue,
    setKeepQueue,
    setSampleRate,
    updateLayerType,
    updateLayerDuty,
    updateLayerVolume,
    updateLayerMidiChannels,
    updateLayerVibratoDepth,
    updateLayerVibratoRate,
    updateLayerOctaveShift,
    toggleCurve,
    addCurvePoint,
    removeSelectedPoint,
    resetCurve,
    selectCurvePoint,
    addLayer,
    removeLayer,
    resetLayers,
    playPreview,
    startCurvePointDrag,
    onPointerMove,
    onPointerUp,
    downloadConvertedFile,
    clearConvertedFiles,
    processQueue,
    cleanup,
  };
}
