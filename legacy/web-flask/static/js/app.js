    const appConfigElement = document.getElementById('octabit-config');
    const appConfig = JSON.parse(appConfigElement.textContent);
    const TRANSLATIONS_BY_LOCALE = appConfig.translationsByLocale || {};
    let TRANSLATIONS = appConfig.translations;
    let CURRENT_LOCALE = appConfig.currentLocale;
    const DEFAULT_LOCALE = appConfig.defaultLocale;
    const SUPPORTED_LOCALES = appConfig.supportedLocales;
    const LOCALE_COOKIE_NAME = appConfig.localeCookieName;
    const ICONS = window.octabitLucideIcons || { svg: () => '' };
    const LOCALE_COOKIE_MAX_AGE_SECONDS = 60 * 60 * 24 * 365;
    const themeController = window.octabitTheme || {};
    const THEME_STORAGE_KEY = themeController.storageKey || 'octabitTheme';
    const THEME_VALUES = ['light', 'dark'];
    const WORKSPACE_API_URL = '/api/workspace';
    const SYNTHESIS_JOBS_API_URL = '/api/synthesis-jobs';
    const CONTROL_SWITCH_TRANSITION_MS = 200;
    const WORKSPACE_CONFIG_SAVE_DELAY_MS = 400;
    const PREVIEW_VOLUME = 0.5;
    const MIN_CURVE_FREQUENCY_HZ = 8.175798915643707;
    const MAX_CURVE_FREQUENCY_HZ = 12543.853951415975;
    const MIN_CURVE_GAIN_DB = -36.0;
    const MAX_CURVE_GAIN_DB = 12.0;
    const MAX_CURVE_POINTS = 8;
    const CURVE_WIDTH = 320;
    const CURVE_HEIGHT = 150;
    const CURVE_MARGIN = { top: 14, right: 14, bottom: 24, left: 38 };
    const curveLogSpan = Math.log(MAX_CURVE_FREQUENCY_HZ) - Math.log(MIN_CURVE_FREQUENCY_HZ);
    const layerPresets = [
        { type: 'pulse', duty: 0.5, volume: 1.0 },
        { type: 'sine', duty: 0.5, volume: 1.0 },
        { type: 'triangle', duty: 0.5, volume: 1.0 },
        { type: 'sawtooth', duty: 0.5, volume: 1.0 },
    ];
    const waveTypeOptions = [
        ['pulse', 'wave.pulse'],
        ['sine', 'wave.sine'],
        ['sawtooth', 'wave.sawtooth'],
        ['triangle', 'wave.triangle'],
    ];
    const maxLayers = layerPresets.length;

    function t(key, params = {}) {
        const template = Object.prototype.hasOwnProperty.call(TRANSLATIONS, key)
            ? TRANSLATIONS[key]
            : key;

        return template.replace(/\{(\w+)\}/g, (match, token) => {
            return Object.prototype.hasOwnProperty.call(params, token) ? params[token] : match;
        });
    }

    function translateStaticSurface() {
        document.title = t('meta.site_title');
        htmlElement.lang = CURRENT_LOCALE;
        document.querySelectorAll('[data-i18n]').forEach((element) => {
            element.textContent = t(element.dataset.i18n);
        });
        document.querySelectorAll('[data-i18n-title]').forEach((element) => {
            element.setAttribute('title', t(element.dataset.i18nTitle));
        });
        document.querySelectorAll('[data-i18n-aria-label]').forEach((element) => {
            element.setAttribute('aria-label', t(element.dataset.i18nAriaLabel));
        });
        document.querySelectorAll('[data-i18n-content]').forEach((element) => {
            element.setAttribute('content', t(element.dataset.i18nContent));
        });
    }

    function createDefaultCurve() {
        return [
            { frequency_hz: MIN_CURVE_FREQUENCY_HZ, gain_db: 0.0 },
            { frequency_hz: MAX_CURVE_FREQUENCY_HZ, gain_db: 0.0 },
        ];
    }

    function createDefaultLayer(index) {
        const preset = layerPresets[index] || layerPresets[0];
        return {
            active: index === 0,
            type: preset.type,
            duty: preset.duty,
            volume: preset.volume,
            curveEnabled: false,
            frequencyCurve: createDefaultCurve(),
            selectedPointIndex: 0,
        };
    }

    let fileQueue = [];
    let layerCount = 1;
    let dragStartIndex;
    let previewAudio = new Audio();
    previewAudio.volume = PREVIEW_VOLUME;
    let dragState = null;
    let layerRenderTimer = null;
    let workspaceConfigSaveTimer = null;
    let isRestoringWorkspace = false;
    let convertedFiles = [];
    const layers = Array.from({ length: maxLayers }, (_, index) => createDefaultLayer(index));

    const synthForm = document.getElementById('synthForm');
    const loading = document.querySelector('.loading');
    const submitBtn = document.getElementById('submitBtn');
    const htmlElement = document.documentElement;
    const dropZone = document.getElementById('dropZone');
    const fileInput = document.getElementById('midi_file');
    const queueList = document.getElementById('queueList');
    const clearQueueBtn = document.getElementById('clearQueueBtn');
    const keepQueueToggle = document.getElementById('keepQueueToggle');
    const queueCountSpan = document.getElementById('queueCount');
    const queueCountAction = document.getElementById('queueCountAction');
    const queueEmpty = document.getElementById('queueEmpty');
    const processingStatus = document.getElementById('processingStatus');
    const themeSelect = document.getElementById('themeSelect');
    const themeSelectIcon = document.getElementById('themeSelectIcon');
    const languageSelect = document.getElementById('languageSelect');
    const languageSelectIcon = document.querySelector('.language-select-icon');
    const githubLinkIcon = document.querySelector('.github-link-icon');
    const layersContainer = document.getElementById('layersContainer');
    const addLayerBtn = document.getElementById('addLayerBtn');
    const removeLayerBtn = document.getElementById('removeLayerBtn');
    const rateSelect = document.getElementById('rate');
    const convertedList = document.getElementById('convertedList');
    const convertedEmpty = document.getElementById('convertedEmpty');
    const convertedCount = document.getElementById('convertedCount');
    const clearConvertedBtn = document.getElementById('clearConvertedBtn');
    const savedKeepQueuePreference = localStorage.getItem('keepQueueAfterSynth');

    document.title = t('meta.site_title');
    htmlElement.setAttribute('lang', CURRENT_LOCALE);
    languageSelect.value = CURRENT_LOCALE;
    if (languageSelectIcon) {
        languageSelectIcon.innerHTML = ICONS.svg('languages', 'lucide-icon language-option-icon');
    }
    if (githubLinkIcon) {
        githubLinkIcon.innerHTML = ICONS.svg('github', 'hugeicon github-link-svg');
    }

    function isThemeValue(value) {
        return THEME_VALUES.includes(value);
    }

    function storedTheme() {
        if (typeof themeController.storedTheme === 'function') {
            return themeController.storedTheme();
        }

        try {
            const value = localStorage.getItem(THEME_STORAGE_KEY);
            return isThemeValue(value) ? value : null;
        } catch (error) {
            return null;
        }
    }

    function clearStoredTheme() {
        try {
            localStorage.removeItem(THEME_STORAGE_KEY);
        } catch (error) {
            console.warn('Failed to clear theme preference.', error);
        }
    }

    function systemTheme() {
        if (typeof themeController.systemTheme === 'function') {
            return themeController.systemTheme();
        }

        if (!window.matchMedia) {
            return 'dark';
        }

        if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
            return 'dark';
        }

        if (window.matchMedia('(prefers-color-scheme: light)').matches) {
            return 'light';
        }

        return 'dark';
    }

    function resolvedTheme() {
        return storedTheme() || systemTheme();
    }

    function isFollowingSystemTheme() {
        return !storedTheme();
    }

    function selectedThemeValue() {
        return storedTheme() || 'system';
    }

    function activeThemeValue() {
        return htmlElement.getAttribute('data-bs-theme') || resolvedTheme();
    }

    function selectedThemeIconName() {
        if (themeSelect.value === 'light') {
            return 'sun';
        }

        if (themeSelect.value === 'dark') {
            return 'moon-star';
        }

        return activeThemeValue() === 'light' ? 'sun' : 'moon-star';
    }

    function syncThemeSelectIcon() {
        if (!themeSelectIcon) {
            return;
        }

        themeSelectIcon.innerHTML = ICONS.svg(selectedThemeIconName(), 'lucide-icon theme-option-icon');
    }

    function applyTheme(theme) {
        if (typeof themeController.applyTheme === 'function') {
            const nextTheme = themeController.applyTheme(theme);
            syncThemeSelectIcon();
            return nextTheme;
        }

        const nextTheme = isThemeValue(theme) ? theme : systemTheme();
        htmlElement.classList.add('theme-change-instant');
        htmlElement.setAttribute('data-bs-theme', nextTheme);
        void htmlElement.offsetHeight;
        htmlElement.classList.remove('theme-change-instant');
        syncThemeSelectIcon();
        return nextTheme;
    }

    function syncThemeSelect() {
        themeSelect.value = selectedThemeValue();
        syncThemeSelectIcon();
    }

    function saveTheme(theme) {
        if (!isThemeValue(theme)) {
            return;
        }

        try {
            localStorage.setItem(THEME_STORAGE_KEY, theme);
        } catch (error) {
            console.warn('Failed to save theme preference.', error);
        }

        applyTheme(theme);
        syncThemeSelect();
    }

    function followSystemTheme() {
        clearStoredTheme();
        applyTheme();
        syncThemeSelect();
    }

    applyTheme(resolvedTheme());
    syncThemeSelect();
    keepQueueToggle.checked = savedKeepQueuePreference === 'true';

    themeSelect.addEventListener('change', () => {
        if (themeSelect.value === 'system') {
            followSystemTheme();
        } else {
            saveTheme(themeSelect.value);
        }
    });

    if (window.matchMedia) {
        const systemThemeQueries = [
            window.matchMedia('(prefers-color-scheme: dark)'),
            window.matchMedia('(prefers-color-scheme: light)'),
        ];
        const syncSystemTheme = () => {
            if (isFollowingSystemTheme()) {
                applyTheme();
                syncThemeSelect();
            }
        };

        systemThemeQueries.forEach((systemThemeQuery) => {
            if (typeof systemThemeQuery.addEventListener === 'function') {
                systemThemeQuery.addEventListener('change', syncSystemTheme);
            } else if (typeof systemThemeQuery.addListener === 'function') {
                systemThemeQuery.addListener(syncSystemTheme);
            }
        });
    }

    languageSelect.addEventListener('change', async () => {
        const selectedLocale = languageSelect.value;
        const nextLocale = SUPPORTED_LOCALES.includes(selectedLocale) ? selectedLocale : DEFAULT_LOCALE;
        if (nextLocale === CURRENT_LOCALE) {
            return;
        }

        languageSelect.disabled = true;
        try {
            TRANSLATIONS = TRANSLATIONS_BY_LOCALE[nextLocale] || TRANSLATIONS_BY_LOCALE[DEFAULT_LOCALE] || TRANSLATIONS;
            CURRENT_LOCALE = nextLocale;
            document.cookie = `${LOCALE_COOKIE_NAME}=${encodeURIComponent(nextLocale)}; path=/; max-age=${LOCALE_COOKIE_MAX_AGE_SECONDS}; SameSite=Lax`;

            const url = new URL(window.location.href);
            if (nextLocale === DEFAULT_LOCALE) {
                url.searchParams.delete('lang');
            } else {
                url.searchParams.set('lang', nextLocale);
            }
            window.history.replaceState({}, '', url.toString());
            translateStaticSurface();
            renderQueue();
            renderConvertedFiles();
            renderLayers();
        } finally {
            languageSelect.disabled = false;
        }
    });

    keepQueueToggle.addEventListener('change', () => {
        localStorage.setItem('keepQueueAfterSynth', keepQueueToggle.checked);
    });

    function isMidiFile(file) {
        return (
            file.type === 'audio/midi'
            || file.type === 'audio/x-midi'
            || /\.midi?$/i.test(file.name)
        );
    }

    function formatFileSize(bytes) {
        if (!Number.isFinite(bytes) || bytes <= 0) {
            return '0 KB';
        }

        if (bytes < 1024 * 1024) {
            return `${Math.max(1, Math.round(bytes / 1024))} KB`;
        }

        return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }

    function currentWorkspaceConfig() {
        return {
            schema: 'octabit.workspace_config.v1',
            sample_rate: Number(rateSelect.value),
            layers: layers.slice(0, layerCount).map((layer) => ({
                type: layer.type,
                duty: Number(layer.duty.toFixed(4)),
                volume: Number(layer.volume.toFixed(4)),
                curve_enabled: Boolean(layer.curveEnabled),
                frequency_curve: layer.frequencyCurve.map((point) => ({
                    frequency_hz: point.frequency_hz,
                    gain_db: Number(point.gain_db.toFixed(4)),
                })),
            })),
        };
    }

    function scheduleWorkspaceConfigSave() {
        if (isRestoringWorkspace) {
            return;
        }
        window.clearTimeout(workspaceConfigSaveTimer);
        workspaceConfigSaveTimer = window.setTimeout(saveWorkspaceConfig, WORKSPACE_CONFIG_SAVE_DELAY_MS);
    }

    async function saveWorkspaceConfig() {
        workspaceConfigSaveTimer = null;
        try {
            await fetch(`${WORKSPACE_API_URL}/config`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(currentWorkspaceConfig()),
            });
        } catch (error) {
            console.warn('Failed to save workspace config.', error);
        }
    }

    function applyWorkspaceConfig(config) {
        if (!config || !Array.isArray(config.layers)) {
            return;
        }

        if ([44100, 48000, 96000].includes(Number(config.sample_rate))) {
            rateSelect.value = String(config.sample_rate);
        }

        layerCount = Math.min(Math.max(config.layers.length, 1), maxLayers);
        for (let index = 0; index < maxLayers; index += 1) {
            layers[index] = createDefaultLayer(index);
        }
        config.layers.slice(0, layerCount).forEach((configLayer, index) => {
            const defaultLayer = createDefaultLayer(index);
            layers[index] = {
                ...defaultLayer,
                type: typeof configLayer.type === 'string' ? configLayer.type : defaultLayer.type,
                duty: Number.isFinite(Number(configLayer.duty)) ? Number(configLayer.duty) : defaultLayer.duty,
                volume: Number.isFinite(Number(configLayer.volume)) ? Number(configLayer.volume) : defaultLayer.volume,
                curveEnabled: Boolean(configLayer.curve_enabled),
                frequencyCurve: Array.isArray(configLayer.frequency_curve) && configLayer.frequency_curve.length
                    ? configLayer.frequency_curve.map((point) => ({
                        frequency_hz: Number(point.frequency_hz),
                        gain_db: Number(point.gain_db),
                    }))
                    : createDefaultCurve(),
                selectedPointIndex: 0,
            };
        });
    }

    function uploadRecordFromApi(upload) {
        return {
            fileId: upload.file_id,
            name: upload.name,
            size: upload.size,
        };
    }

    function convertedRecordFromApi(file) {
        return {
            jobId: file.job_id,
            name: file.name,
            sourceName: file.source_name,
            size: file.size,
            url: new URL(file.download_url, window.location.origin).toString(),
            objectUrl: false,
            deleteUrl: file.delete_url ? new URL(file.delete_url, window.location.origin).toString() : null,
        };
    }

    async function restoreWorkspace() {
        try {
            const response = await fetch(WORKSPACE_API_URL);
            const payload = await readJsonResponse(response);
            if (!response.ok) {
                throw new Error(responseErrorMessage(payload, response.statusText));
            }

            isRestoringWorkspace = true;
            fileQueue = Array.isArray(payload.uploads) ? payload.uploads.map(uploadRecordFromApi) : [];
            convertedFiles = Array.isArray(payload.converted_files)
                ? payload.converted_files.map(convertedRecordFromApi)
                : [];
            applyWorkspaceConfig(payload.config);
        } catch (error) {
            console.warn('Failed to restore workspace.', error);
        } finally {
            isRestoringWorkspace = false;
            renderQueue();
            renderConvertedFiles();
            renderLayers();
        }
    }

    async function persistQueueOrder() {
        try {
            await fetch(`${WORKSPACE_API_URL}/queue`, {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    file_ids: fileQueue.map((file) => file.fileId),
                }),
            });
        } catch (error) {
            console.warn('Failed to save queue order.', error);
        }
    }

    function renderQueue() {
        queueList.innerHTML = '';
        fileQueue.forEach((file, index) => {
            const li = document.createElement('li');
            li.className = 'queue-item';
            li.setAttribute('draggable', 'true');
            li.setAttribute('data-index', index);
            li.setAttribute('data-full-name', file.name);

            const fileInfo = document.createElement('div');
            fileInfo.className = 'min-w-0';

            const fileName = document.createElement('div');
            fileName.className = 'file-name';
            fileName.textContent = file.name;

            const fileMeta = document.createElement('div');
            fileMeta.className = 'file-meta';
            fileMeta.textContent = formatFileSize(file.size);

            const removeButton = document.createElement('button');
            removeButton.type = 'button';
            removeButton.className = 'remove-btn';
            removeButton.innerHTML = ICONS.svg('x');
            removeButton.setAttribute('aria-label', t('queue.remove_file', { filename: file.name }));
            removeButton.addEventListener('click', () => window.removeFromQueue(index));

            fileInfo.append(fileName, fileMeta);
            li.append(fileInfo, removeButton);
            queueList.appendChild(li);
        });
        queueCountSpan.textContent = fileQueue.length;
        queueCountAction.textContent = fileQueue.length;
        queueEmpty.style.display = fileQueue.length > 0 ? 'none' : 'grid';
        clearQueueBtn.style.display = fileQueue.length > 0 ? 'block' : 'none';
        submitBtn.disabled = fileQueue.length === 0;
    }

    function renderConvertedFiles() {
        convertedList.innerHTML = '';
        convertedFiles.forEach((file, index) => {
            const li = document.createElement('li');
            li.className = 'converted-item';

            const fileInfo = document.createElement('div');
            fileInfo.className = 'min-w-0';

            const fileName = document.createElement('div');
            fileName.className = 'file-name';
            fileName.textContent = file.name;

            const fileMeta = document.createElement('div');
            fileMeta.className = 'file-meta';
            fileMeta.textContent = `${formatFileSize(file.size)} / ${file.sourceName}`;

            const downloadButton = document.createElement('button');
            downloadButton.type = 'button';
            downloadButton.className = 'download-btn';
            downloadButton.textContent = t('converted.download');
            downloadButton.addEventListener('click', () => downloadConvertedFile(index));

            fileInfo.append(fileName, fileMeta);
            li.append(fileInfo, downloadButton);
            convertedList.appendChild(li);
        });

        convertedCount.textContent = convertedFiles.length;
        convertedEmpty.style.display = convertedFiles.length > 0 ? 'none' : 'grid';
        clearConvertedBtn.style.display = convertedFiles.length > 0 ? 'block' : 'none';
    }

    function downloadConvertedFile(index) {
        const convertedFile = convertedFiles[index];
        if (!convertedFile) {
            return;
        }

        const anchor = document.createElement('a');
        anchor.href = convertedFile.url;
        anchor.download = convertedFile.name;
        document.body.appendChild(anchor);
        anchor.click();
        document.body.removeChild(anchor);
    }

    function addConvertedFile(downloadName, blob, sourceName) {
        convertedFiles.unshift({
            name: downloadName,
            sourceName,
            size: blob.size,
            url: window.URL.createObjectURL(blob),
            objectUrl: true,
        });
        renderConvertedFiles();
    }

    function addConvertedServerFile(downloadName, size, sourceName, downloadUrl, deleteUrl, jobId = null) {
        if (jobId) {
            convertedFiles = convertedFiles.filter((file) => file.jobId !== jobId);
        }
        convertedFiles.unshift({
            jobId,
            name: downloadName,
            sourceName,
            size,
            url: downloadUrl,
            objectUrl: false,
            deleteUrl,
        });
        renderConvertedFiles();
    }

    function releaseConvertedServerFile(file) {
        if (!file.deleteUrl) {
            return Promise.resolve();
        }

        return fetch(file.deleteUrl, {
            method: 'DELETE',
            keepalive: true,
        }).catch((error) => {
            console.warn('Failed to delete converted server file.', error);
        });
    }

    async function clearConvertedFiles() {
        const filesToClear = [...convertedFiles];
        convertedFiles = [];
        renderConvertedFiles();

        filesToClear.forEach((file) => {
            if (file.objectUrl) {
                window.URL.revokeObjectURL(file.url);
            }
        });

        await Promise.all(filesToClear.map(releaseConvertedServerFile));
    }

    queueList.addEventListener('dragstart', (event) => {
        const item = event.target.closest('.queue-item');
        if (!item) return;
        dragStartIndex = parseInt(item.dataset.index, 10);
        item.classList.add('dragging');
    });

    queueList.addEventListener('dragend', (event) => {
        const item = event.target.closest('.queue-item');
        if (item) item.classList.remove('dragging');
    });

    queueList.addEventListener('dragover', (event) => {
        event.preventDefault();
        const target = event.target.closest('.queue-item');
        if (!target) return;
        document.querySelectorAll('.queue-item').forEach(item => item.classList.remove('drag-over'));
        target.classList.add('drag-over');
    });

    queueList.addEventListener('dragleave', (event) => {
        const target = event.target.closest('.queue-item');
        if (target) target.classList.remove('drag-over');
    });

    queueList.addEventListener('drop', (event) => {
        event.preventDefault();
        const target = event.target.closest('.queue-item');
        if (!target) return;
        target.classList.remove('drag-over');
        const dragEndIndex = parseInt(target.dataset.index, 10);
        const [draggedItem] = fileQueue.splice(dragStartIndex, 1);
        fileQueue.splice(dragEndIndex, 0, draggedItem);
        renderQueue();
        persistQueueOrder();
    });

    async function uploadWorkspaceFile(file) {
        const formData = new FormData();
        formData.append('midi_file', file);
        const response = await fetch(`${WORKSPACE_API_URL}/uploads`, {
            method: 'POST',
            body: formData,
        });
        const payload = await readJsonResponse(response);
        if (!response.ok) {
            throw new Error(responseErrorMessage(payload, response.statusText));
        }
        return uploadRecordFromApi(payload.upload);
    }

    async function addToQueue(files) {
        const uploadedFiles = [];
        for (const file of files) {
            if (!isMidiFile(file)) {
                continue;
            }
            try {
                uploadedFiles.push(await uploadWorkspaceFile(file));
            } catch (error) {
                alert(t('alerts.upload_error', {
                    filename: file.name,
                    error: error.message || t('alerts.processing_unknown', { filename: file.name }),
                }));
            }
        }
        fileQueue.push(...uploadedFiles);
        renderQueue();
        fileInput.value = '';
    }

    async function releaseWorkspaceUpload(file) {
        if (!file.fileId) {
            return;
        }
        await fetch(`${WORKSPACE_API_URL}/uploads/${file.fileId}`, {
            method: 'DELETE',
            keepalive: true,
        });
    }

    window.removeFromQueue = async (index) => {
        const [file] = fileQueue.splice(index, 1);
        renderQueue();
        try {
            await releaseWorkspaceUpload(file);
        } catch (error) {
            console.warn('Failed to delete workspace upload.', error);
        }
    };

    clearQueueBtn.addEventListener('click', async () => {
        const filesToClear = [...fileQueue];
        fileQueue = [];
        renderQueue();
        fileInput.value = '';
        await Promise.all(filesToClear.map((file) => (
            releaseWorkspaceUpload(file).catch((error) => {
                console.warn('Failed to delete workspace upload.', error);
            })
        )));
    });

    clearConvertedBtn.addEventListener('click', () => {
        if (window.confirm(t('converted.clear_confirm'))) {
            clearConvertedFiles();
        }
    });

    dropZone.addEventListener('dragover', (event) => {
        event.preventDefault();
        dropZone.classList.add('dragover');
    });

    ['dragleave', 'dragend'].forEach((type) => {
        dropZone.addEventListener(type, () => dropZone.classList.remove('dragover'));
    });

    dropZone.addEventListener('drop', (event) => {
        event.preventDefault();
        dropZone.classList.remove('dragover');
        if (event.dataTransfer.files.length) addToQueue(event.dataTransfer.files);
    });

    fileInput.addEventListener('change', () => addToQueue(fileInput.files));

    function clamp(value, min, max) {
        return Math.min(Math.max(value, min), max);
    }

    function formatFrequency(value) {
        return value >= 1000 ? `${(value / 1000).toFixed(2)} kHz` : `${value.toFixed(1)} Hz`;
    }

    function formatGainDb(value) {
        return `${value >= 0 ? '+' : ''}${value.toFixed(1)} dB`;
    }

    function plotWidth() {
        return CURVE_WIDTH - CURVE_MARGIN.left - CURVE_MARGIN.right;
    }

    function plotHeight() {
        return CURVE_HEIGHT - CURVE_MARGIN.top - CURVE_MARGIN.bottom;
    }

    function frequencyToX(frequencyHz) {
        const ratio = (Math.log(frequencyHz) - Math.log(MIN_CURVE_FREQUENCY_HZ)) / curveLogSpan;
        return CURVE_MARGIN.left + (ratio * plotWidth());
    }

    function xToFrequency(x) {
        const ratio = clamp((x - CURVE_MARGIN.left) / plotWidth(), 0, 1);
        return MIN_CURVE_FREQUENCY_HZ * ((MAX_CURVE_FREQUENCY_HZ / MIN_CURVE_FREQUENCY_HZ) ** ratio);
    }

    function gainToY(gainDb) {
        const ratio = (MAX_CURVE_GAIN_DB - gainDb) / (MAX_CURVE_GAIN_DB - MIN_CURVE_GAIN_DB);
        return CURVE_MARGIN.top + (ratio * plotHeight());
    }

    function yToGain(y) {
        const ratio = clamp((y - CURVE_MARGIN.top) / plotHeight(), 0, 1);
        return MAX_CURVE_GAIN_DB - (ratio * (MAX_CURVE_GAIN_DB - MIN_CURVE_GAIN_DB));
    }

    function evaluateCurveGainDb(points, frequencyHz) {
        if (!points.length) return 0.0;
        if (frequencyHz <= points[0].frequency_hz) return points[0].gain_db;
        if (frequencyHz >= points[points.length - 1].frequency_hz) return points[points.length - 1].gain_db;
        for (let index = 0; index < points.length - 1; index += 1) {
            const leftPoint = points[index];
            const rightPoint = points[index + 1];
            if (frequencyHz >= leftPoint.frequency_hz && frequencyHz <= rightPoint.frequency_hz) {
                const leftLog = Math.log(leftPoint.frequency_hz);
                const rightLog = Math.log(rightPoint.frequency_hz);
                const frequencyLog = Math.log(frequencyHz);
                const ratio = (frequencyLog - leftLog) / (rightLog - leftLog);
                return leftPoint.gain_db + (ratio * (rightPoint.gain_db - leftPoint.gain_db));
            }
        }
        return points[points.length - 1].gain_db;
    }

    function activeLayerTypes(excludedLayerIndex = null) {
        return new Set(layers.slice(0, layerCount)
            .filter((layer, index) => index !== excludedLayerIndex)
            .map((layer) => layer.type));
    }

    function firstUnusedWaveType() {
        const usedTypes = activeLayerTypes();
        const option = waveTypeOptions.find(([value]) => !usedTypes.has(value));
        return option ? option[0] : null;
    }

    function createWaveOptions(currentType, layerIndex) {
        const usedTypes = activeLayerTypes(layerIndex);
        return waveTypeOptions.map(([value, labelKey]) => `
            <option
                value="${value}"
                ${currentType === value ? 'selected' : ''}
                ${usedTypes.has(value) ? 'disabled' : ''}
            >${t(labelKey)}</option>
        `).join('');
    }

    function buildCurvePath(points) {
        return points.map((point, index) => {
            const command = index === 0 ? 'M' : 'L';
            return `${command} ${frequencyToX(point.frequency_hz).toFixed(2)} ${gainToY(point.gain_db).toFixed(2)}`;
        }).join(' ');
    }

    function buildCurveArea(points) {
        const startX = frequencyToX(points[0].frequency_hz).toFixed(2);
        const endX = frequencyToX(points[points.length - 1].frequency_hz).toFixed(2);
        const bottomY = (CURVE_MARGIN.top + plotHeight()).toFixed(2);
        return `M ${startX} ${bottomY} ${buildCurvePath(points).slice(2)} L ${endX} ${bottomY} Z`;
    }

    function buildCurveSvg(layer, layerIndex) {
        const selectedPoint = layer.frequencyCurve[layer.selectedPointIndex] || layer.frequencyCurve[0];
        const gainTicks = [MAX_CURVE_GAIN_DB, 0, MIN_CURVE_GAIN_DB];
        const freqTicks = [MIN_CURVE_FREQUENCY_HZ, 27.5, 110.0, 440.0, 1760.0, MAX_CURVE_FREQUENCY_HZ];

        return `
            <div class="curve-summary">
                ${t('curve.selected_point', {
                    frequency: formatFrequency(selectedPoint.frequency_hz),
                    gain: formatGainDb(selectedPoint.gain_db),
                })}
            </div>
            <svg
                class="curve-svg"
                id="curveSvg${layerIndex}"
                viewBox="0 0 ${CURVE_WIDTH} ${CURVE_HEIGHT}"
                aria-label="${t('curve.aria_label', { index: layerIndex + 1 })}"
            >
                <rect
                    x="${CURVE_MARGIN.left}"
                    y="${CURVE_MARGIN.top}"
                    width="${plotWidth()}"
                    height="${plotHeight()}"
                    fill="transparent"
                ></rect>
                ${gainTicks.map((gainDb) => `
                    <g>
                        <line
                            class="${gainDb === 0 ? 'curve-zero-line' : 'curve-grid-line'}"
                            x1="${CURVE_MARGIN.left}"
                            y1="${gainToY(gainDb)}"
                            x2="${CURVE_MARGIN.left + plotWidth()}"
                            y2="${gainToY(gainDb)}"
                        ></line>
                        <text class="curve-axis-label" x="4" y="${gainToY(gainDb) + 4}">${formatGainDb(gainDb)}</text>
                    </g>
                `).join('')}
                ${freqTicks.map((frequencyHz) => `
                    <g>
                        <line
                            class="curve-grid-line"
                            x1="${frequencyToX(frequencyHz)}"
                            y1="${CURVE_MARGIN.top}"
                            x2="${frequencyToX(frequencyHz)}"
                            y2="${CURVE_MARGIN.top + plotHeight()}"
                        ></line>
                        <text
                            class="curve-axis-label"
                            x="${frequencyToX(frequencyHz)}"
                            y="${CURVE_HEIGHT - 6}"
                            text-anchor="middle"
                        >${frequencyHz >= 1000 ? `${(frequencyHz / 1000).toFixed(1)}k` : Math.round(frequencyHz)}</text>
                    </g>
                `).join('')}
                <path class="curve-fill" d="${buildCurveArea(layer.frequencyCurve)}"></path>
                <path class="curve-path" d="${buildCurvePath(layer.frequencyCurve)}"></path>
                ${layer.frequencyCurve.map((point, pointIndex) => {
                    const isEndpoint = pointIndex === 0 || pointIndex === layer.frequencyCurve.length - 1;
                    const pointRadius = isEndpoint ? 3.4 : 3.0;

                    return `
                        <circle
                            class="curve-point-hit"
                            cx="${frequencyToX(point.frequency_hz)}"
                            cy="${gainToY(point.gain_db)}"
                            r="7"
                            onpointerdown="startCurvePointDrag(${layerIndex}, ${pointIndex}, event)"
                            onclick="selectCurvePoint(${layerIndex}, ${pointIndex})"
                        ></circle>
                        <circle
                            class="curve-point ${layer.selectedPointIndex === pointIndex ? 'selected' : ''}"
                            cx="${frequencyToX(point.frequency_hz)}"
                            cy="${gainToY(point.gain_db)}"
                            r="${pointRadius}"
                        ></circle>
                    `;
                }).join('')}
            </svg>
        `;
    }

    function faderFillPercent(value, min, max) {
        const ratio = (Number(value) - min) / (max - min);
        return `${clamp(ratio * 100, 0, 100).toFixed(2)}%`;
    }

    function createFaderScale(ticks, min, max) {
        return ticks.map((tick, index) => {
            const position = faderFillPercent(tick.value, min, max);
            const edgeClass = index === 0
                ? 'is-start'
                : index === ticks.length - 1
                    ? 'is-end'
                    : '';

            return `
                <span
                    class="fader-scale-mark ${edgeClass}"
                    style="--tick-position: ${position}"
                    aria-hidden="true"
                ></span>
                <span
                    class="fader-scale-label ${edgeClass}"
                    style="--tick-position: ${position}"
                >${tick.label}</span>
            `;
        }).join('');
    }

    function updateFaderFill(input) {
        if (!input) {
            return;
        }

        const min = Number(input.min);
        const max = Number(input.max);
        input.style.setProperty('--fill', faderFillPercent(input.value, min, max));
    }

    function renderLayers() {
        layersContainer.innerHTML = '';

        for (let layerIndex = 0; layerIndex < layerCount; layerIndex += 1) {
            const layer = layers[layerIndex];
            const selectedPoint = layer.frequencyCurve[layer.selectedPointIndex] || layer.frequencyCurve[0];
            const canRemoveSelected = layer.frequencyCurve.length > 2
                && layer.selectedPointIndex > 0
                && layer.selectedPointIndex < layer.frequencyCurve.length - 1;

            const card = document.createElement('div');
            card.className = 'layer-card';
            card.innerHTML = `
                <div class="layer-title-row">
                    <div>
                        <div class="layer-title">${t('layer.title', { index: layerIndex + 1 })}</div>
                    </div>
                    <button
                        type="button"
                        class="preview-btn"
                        onclick="playPreview(${layerIndex})"
                        title="${t('layer.play_preview')}"
                        aria-label="${t('layer.play_preview')}"
                    >${ICONS.svg('play')}</button>
                </div>
                <div class="layer-control-grid">
                    <div class="field-block waveform-field">
                        <label class="field-label" for="waveType${layerIndex}">${t('layer.waveform_type')}</label>
                        <select
                            class="form-select control-select"
                            id="waveType${layerIndex}"
                            onchange="updateLayerType(${layerIndex}, this.value)"
                        >
                            ${createWaveOptions(layer.type, layerIndex)}
                        </select>
                    </div>
                    <div class="field-block" style="display: ${layer.type === 'pulse' ? 'grid' : 'none'};">
                        <label class="fader-label" for="dutyFader${layerIndex}">
                            <span>${t('layer.pulse_width')}</span>
                            <input
                                type="number"
                                class="readout"
                                id="dutyValue${layerIndex}"
                                min="0.01"
                                max="0.99"
                                step="0.01"
                                value="${layer.duty.toFixed(2)}"
                                inputmode="decimal"
                                onchange="updateLayerDuty(${layerIndex}, this.value, this)"
                            >
                        </label>
                        <div class="fader-shell">
                            <input
                                type="range"
                                class="fader-input"
                                id="dutyFader${layerIndex}"
                                min="0.01"
                                max="0.99"
                                step="0.01"
                                value="${layer.duty}"
                                style="--fill: ${faderFillPercent(layer.duty, 0.01, 0.99)}"
                                oninput="updateLayerDuty(${layerIndex}, this.value, this)"
                            >
                        </div>
                        <div class="fader-scale" aria-hidden="true">
                            ${createFaderScale([
                                { value: 0.01, label: '0.01' },
                                { value: 0.25, label: '0.25' },
                                { value: 0.50, label: '0.50' },
                                { value: 0.75, label: '0.75' },
                                { value: 0.99, label: '0.99' },
                            ], 0.01, 0.99)}
                        </div>
                    </div>
                    <div class="field-block layer-volume-control ${layer.type === 'pulse' ? '' : 'layer-volume-wide'}">
                        <label class="fader-label" for="volumeFader${layerIndex}">
                            <span>${t('layer.base_volume')}</span>
                            <input
                                type="number"
                                class="readout"
                                id="volumeValue${layerIndex}"
                                min="0.00"
                                max="2.00"
                                step="0.01"
                                value="${layer.volume.toFixed(2)}"
                                inputmode="decimal"
                                onchange="updateLayerVolume(${layerIndex}, this.value, this)"
                            >
                        </label>
                        <div class="fader-shell">
                            <input
                                type="range"
                                class="fader-input"
                                id="volumeFader${layerIndex}"
                                min="0.0"
                                max="2.0"
                                step="0.01"
                                value="${layer.volume}"
                                style="--fill: ${faderFillPercent(layer.volume, 0.0, 2.0)}"
                                oninput="updateLayerVolume(${layerIndex}, this.value, this)"
                            >
                        </div>
                        <div class="fader-scale" aria-hidden="true">
                            ${createFaderScale([
                                { value: 0.00, label: '0.00' },
                                { value: 0.50, label: '0.50' },
                                { value: 1.00, label: '1.00' },
                                { value: 1.50, label: '1.50' },
                                { value: 2.00, label: '2.00' },
                            ], 0.0, 2.0)}
                        </div>
                    </div>
                    <div class="control-switch layer-curve-toggle">
                        <input
                            class="control-switch-input"
                            type="checkbox"
                            id="curveToggle${layerIndex}"
                            ${layer.curveEnabled ? 'checked' : ''}
                            onchange="toggleCurveEnabled(${layerIndex}, this.checked)"
                        >
                        <label class="control-switch-label" for="curveToggle${layerIndex}">
                            <span class="control-switch-track" aria-hidden="true">
                                <span class="control-switch-thumb"></span>
                            </span>
                            <span class="control-switch-text">${t('layer.enable_curve')}</span>
                        </label>
                    </div>
                </div>
                ${layer.curveEnabled ? `
                    <div class="curve-panel">
                        <div class="curve-toolbar">
                            <button
                                type="button"
                                class="utility-btn"
                                onclick="addCurvePoint(${layerIndex})"
                                ${layer.frequencyCurve.length >= MAX_CURVE_POINTS ? 'disabled' : ''}
                            >
                                ${t('curve.add_point')}
                            </button>
                            <button
                                type="button"
                                class="utility-btn"
                                onclick="removeSelectedPoint(${layerIndex})"
                                ${canRemoveSelected ? '' : 'disabled'}
                            >
                                ${t('curve.remove_selected')}
                            </button>
                            <button
                                type="button"
                                class="utility-btn"
                                onclick="resetCurve(${layerIndex})"
                            >
                                ${t('curve.reset')}
                            </button>
                        </div>
                        <div class="curve-summary">
                            ${t('curve.drag_help')}
                        </div>
                        ${buildCurveSvg(layer, layerIndex)}
                        <div class="curve-summary mt-2">
                            ${t('curve.points_summary', {
                                count: layer.frequencyCurve.length,
                                frequency: formatFrequency(selectedPoint.frequency_hz),
                                gain: formatGainDb(selectedPoint.gain_db),
                            })}
                        </div>
                    </div>
                ` : ''}
            `;
            layersContainer.appendChild(card);
        }

        updateLayerButtons();
    }

    function updateLayerButtons() {
        addLayerBtn.disabled = layerCount === maxLayers || !firstUnusedWaveType();
        removeLayerBtn.disabled = layerCount === 1;
        removeLayerBtn.style.display = 'inline-flex';
    }

    function activeLayersPayload() {
        return layers.slice(0, layerCount).map((layer) => ({
            type: layer.type,
            duty: Number(layer.duty.toFixed(4)),
            volume: Number(layer.volume.toFixed(4)),
            frequency_curve: layer.curveEnabled
                ? layer.frequencyCurve.map((point) => ({
                    frequency_hz: point.frequency_hz,
                    gain_db: Number(point.gain_db.toFixed(4)),
                }))
                : [],
        }));
    }

    function updateLayerType(layerIndex, value) {
        const validWaveType = waveTypeOptions.some(([optionValue]) => optionValue === value);
        if (!validWaveType || activeLayerTypes(layerIndex).has(value)) {
            renderLayers();
            return;
        }
        layers[layerIndex].type = value;
        renderLayers();
        scheduleWorkspaceConfigSave();
    }

    function normaliseDecimalInput(value, min, max) {
        const parsedValue = parseFloat(value);
        const finiteValue = Number.isFinite(parsedValue) ? parsedValue : min;
        return Number(clamp(finiteValue, min, max).toFixed(2));
    }

    function updateLayerDuty(layerIndex, value, input = null) {
        const duty = normaliseDecimalInput(value, 0.01, 0.99);
        layers[layerIndex].duty = duty;
        const dutyValue = document.getElementById(`dutyValue${layerIndex}`);
        if (dutyValue) {
            dutyValue.value = duty.toFixed(2);
        }
        const dutyFader = document.getElementById(`dutyFader${layerIndex}`);
        if (dutyFader) {
            dutyFader.value = duty.toFixed(2);
        }
        updateFaderFill(dutyFader || input);
        scheduleWorkspaceConfigSave();
    }

    function updateLayerVolume(layerIndex, value, input = null) {
        const volume = normaliseDecimalInput(value, 0.0, 2.0);
        layers[layerIndex].volume = volume;
        const volumeValue = document.getElementById(`volumeValue${layerIndex}`);
        if (volumeValue) {
            volumeValue.value = volume.toFixed(2);
        }
        const volumeFader = document.getElementById(`volumeFader${layerIndex}`);
        if (volumeFader) {
            volumeFader.value = volume.toFixed(2);
        }
        updateFaderFill(volumeFader || input);
        scheduleWorkspaceConfigSave();
    }

    function toggleCurveEnabled(layerIndex, enabled) {
        const layer = layers[layerIndex];
        layer.curveEnabled = enabled;
        if (!layer.frequencyCurve.length) {
            layer.frequencyCurve = createDefaultCurve();
            layer.selectedPointIndex = 0;
        }
        window.clearTimeout(layerRenderTimer);
        layerRenderTimer = window.setTimeout(() => {
            layerRenderTimer = null;
            renderLayers();
            scheduleWorkspaceConfigSave();
        }, CONTROL_SWITCH_TRANSITION_MS);
    }

    function selectCurvePoint(layerIndex, pointIndex) {
        layers[layerIndex].selectedPointIndex = pointIndex;
        renderLayers();
    }

    function addCurvePoint(layerIndex) {
        const layer = layers[layerIndex];
        if (layer.frequencyCurve.length >= MAX_CURVE_POINTS) return;

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
        renderLayers();
        scheduleWorkspaceConfigSave();
    }

    function removeSelectedPoint(layerIndex) {
        const layer = layers[layerIndex];
        if (layer.selectedPointIndex <= 0 || layer.selectedPointIndex >= layer.frequencyCurve.length - 1) {
            return;
        }

        layer.frequencyCurve.splice(layer.selectedPointIndex, 1);
        layer.selectedPointIndex = Math.max(0, layer.selectedPointIndex - 1);
        renderLayers();
        scheduleWorkspaceConfigSave();
    }

    function resetCurve(layerIndex) {
        layers[layerIndex].frequencyCurve = createDefaultCurve();
        layers[layerIndex].selectedPointIndex = 0;
        renderLayers();
        scheduleWorkspaceConfigSave();
    }

    window.updateLayerType = updateLayerType;
    window.updateLayerDuty = updateLayerDuty;
    window.updateLayerVolume = updateLayerVolume;
    window.toggleCurveEnabled = toggleCurveEnabled;
    window.addCurvePoint = addCurvePoint;
    window.removeSelectedPoint = removeSelectedPoint;
    window.resetCurve = resetCurve;
    window.selectCurvePoint = selectCurvePoint;

    window.startCurvePointDrag = (layerIndex, pointIndex, event) => {
        event.preventDefault();
        layers[layerIndex].selectedPointIndex = pointIndex;
        dragState = { layerIndex, pointIndex };
        renderLayers();
    };

    window.addLayer = () => {
        if (layerCount >= maxLayers) return;
        const unusedType = firstUnusedWaveType();
        if (!unusedType) return;
        layers[layerCount] = createDefaultLayer(layerCount);
        layers[layerCount].type = unusedType;
        layerCount += 1;
        layers[layerCount - 1].active = true;
        renderLayers();
        scheduleWorkspaceConfigSave();
    };

    window.removeLayer = () => {
        if (layerCount <= 1) return;
        layers[layerCount - 1] = createDefaultLayer(layerCount - 1);
        layerCount -= 1;
        renderLayers();
        scheduleWorkspaceConfigSave();
    };

    window.playPreview = (layerIndex) => {
        const layer = layers[layerIndex];
        let src = `${layer.type}.wav`;
        if (layer.type === 'pulse') {
            src = `pulse_${layer.duty < 0.18 ? '10' : layer.duty < 0.38 ? '25' : '50'}.wav`;
        }
        previewAudio.src = `/static/previews/${src}`;
        previewAudio.play().catch((error) => console.error('Preview failed:', error));
    };

    addLayerBtn.addEventListener('click', window.addLayer);
    removeLayerBtn.addEventListener('click', window.removeLayer);
    rateSelect.addEventListener('change', scheduleWorkspaceConfigSave);

    layersContainer.addEventListener('pointerdown', (event) => {
        const fader = event.target.closest('.fader-input');
        if (fader) {
            fader.classList.add('is-dragging');
        }
    });

    window.addEventListener('pointermove', (event) => {
        if (!dragState) return;

        const { layerIndex, pointIndex } = dragState;
        const svg = document.getElementById(`curveSvg${layerIndex}`);
        if (!svg) return;

        const rect = svg.getBoundingClientRect();
        const localX = ((event.clientX - rect.left) / rect.width) * CURVE_WIDTH;
        const localY = ((event.clientY - rect.top) / rect.height) * CURVE_HEIGHT;
        const layer = layers[layerIndex];
        const points = layer.frequencyCurve;
        const point = points[pointIndex];

        point.gain_db = Number(clamp(yToGain(localY), MIN_CURVE_GAIN_DB, MAX_CURVE_GAIN_DB).toFixed(4));
        if (pointIndex === 0) {
            point.frequency_hz = MIN_CURVE_FREQUENCY_HZ;
        } else if (pointIndex === points.length - 1) {
            point.frequency_hz = MAX_CURVE_FREQUENCY_HZ;
        } else {
            const minFrequency = points[pointIndex - 1].frequency_hz * 1.0001;
            const maxFrequency = points[pointIndex + 1].frequency_hz / 1.0001;
            point.frequency_hz = clamp(xToFrequency(localX), minFrequency, maxFrequency);
        }

        renderLayers();
        scheduleWorkspaceConfigSave();
    });

    window.addEventListener('pointerup', () => {
        dragState = null;
        document.querySelectorAll('.fader-input.is-dragging').forEach((fader) => {
            fader.classList.remove('is-dragging');
        });
    });

    function extractDownloadName(response, fallbackName) {
        const disposition = response.headers.get('Content-Disposition') || '';
        const utfMatch = disposition.match(/filename\*=UTF-8''([^;]+)/i);
        if (utfMatch) return decodeURIComponent(utfMatch[1]);
        const plainMatch = disposition.match(/filename="?([^"]+)"?/i);
        return plainMatch ? plainMatch[1] : fallbackName;
    }

    function sleep(ms) {
        return new Promise((resolve) => window.setTimeout(resolve, ms));
    }

    async function readJsonResponse(response) {
        try {
            return await response.json();
        } catch (error) {
            return {};
        }
    }

    function responseErrorMessage(payload, fallbackMessage) {
        if (!payload || typeof payload !== 'object') {
            return fallbackMessage;
        }

        if (payload.error && typeof payload.error === 'object') {
            return payload.error.message || payload.error.code || fallbackMessage;
        }

        if (typeof payload.error === 'string') {
            return payload.error;
        }

        return fallbackMessage;
    }

    async function waitForSynthesiseJob(jobId, file, index, total) {
        while (true) {
            const response = await fetch(`${SYNTHESIS_JOBS_API_URL}/${jobId}`);
            const payload = await readJsonResponse(response);
            if (!response.ok && !['ready', 'failed', 'expired'].includes(payload.status)) {
                throw new Error(responseErrorMessage(payload, response.statusText));
            }

            if (payload.status === 'ready') {
                processingStatus.textContent = t('status.file_ready', {
                    current: index + 1,
                    total,
                    filename: file.name,
                });
                return payload;
            }

            if (payload.status === 'failed' || payload.status === 'expired') {
                throw new Error(responseErrorMessage(payload, payload.status));
            }

            processingStatus.textContent = t('status.rendering_file', {
                current: index + 1,
                total,
                filename: file.name,
            });
            await sleep(1000);
        }
    }

    synthForm.onsubmit = async (event) => {
        event.preventDefault();
        loading.classList.add('is-visible');
        submitBtn.disabled = true;

        const filesToProcess = [...fileQueue];
        const failedFiles = [];
        const config = currentWorkspaceConfig();
        for (let index = 0; index < filesToProcess.length; index += 1) {
            const file = filesToProcess[index];
            processingStatus.textContent = t('status.processing_file', {
                current: index + 1,
                total: filesToProcess.length,
                filename: file.name,
            });

            try {
                const response = await fetch(SYNTHESIS_JOBS_API_URL, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        file_id: file.fileId,
                        config,
                    }),
                });
                if (!response.ok) {
                    const errorPayload = await readJsonResponse(response);
                    failedFiles.push(file);
                    alert(t('alerts.processing_error', {
                        filename: file.name,
                        error: responseErrorMessage(errorPayload, response.statusText),
                    }));
                    continue;
                }

                const job = await readJsonResponse(response);
                if (!job.job_id) {
                    throw new Error(responseErrorMessage(job, response.statusText));
                }
                const readyJob = job.status === 'ready'
                    ? job
                    : await waitForSynthesiseJob(job.job_id, file, index, filesToProcess.length);
                const downloadUrl = new URL(readyJob.download_url, window.location.origin).toString();
                addConvertedServerFile(
                    readyJob.download_name || `${file.name.replace(/\.[^.]+$/, '') || 'output'}.wav`,
                    readyJob.size_bytes || 0,
                    file.name,
                    downloadUrl,
                    readyJob.delete_url ? new URL(readyJob.delete_url, window.location.origin).toString() : null,
                    readyJob.job_id || null,
                );
                processingStatus.textContent = t('status.downloading_file', {
                    current: index + 1,
                    total: filesToProcess.length,
                    filename: file.name,
                });
                downloadConvertedFile(0);
            } catch (error) {
                failedFiles.push(file);
                alert(t('alerts.processing_error', {
                    filename: file.name,
                    error: error.message || t('alerts.processing_unknown', { filename: file.name }),
                }));
            }
        }

        loading.classList.remove('is-visible');
        submitBtn.disabled = false;
        processingStatus.textContent = t('status.generating_audio');
        if (!keepQueueToggle.checked) {
            const failedFileIds = new Set(failedFiles.map((file) => file.fileId));
            const processedFiles = filesToProcess.filter((file) => !failedFileIds.has(file.fileId));
            await Promise.all(processedFiles.map((file) => (
                releaseWorkspaceUpload(file).catch((error) => {
                    console.warn('Failed to delete processed workspace upload.', error);
                })
            )));
            fileQueue = failedFiles;
        }
        renderQueue();
        fileInput.value = '';
    };

    function initialisePage() {
        translateStaticSurface();
        renderQueue();
        renderConvertedFiles();
        renderLayers();
        restoreWorkspace();
    }

    initialisePage();
