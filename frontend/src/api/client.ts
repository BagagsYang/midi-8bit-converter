import type {
  CreateSynthesisJobRequest,
  SynthesisJobResponse,
  WorkspaceConfigResponse,
  WorkspaceConfigV1,
  WorkspaceQueueResponse,
  WorkspaceStateResponse,
  WorkspaceUploadResponse,
} from '../types/api';

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return {};
  }
}

export function responseErrorMessage(payload: unknown, fallbackMessage: string): string {
  if (!payload || typeof payload !== 'object') {
    return fallbackMessage;
  }

  const record = payload as { error?: unknown };
  if (record.error && typeof record.error === 'object') {
    const error = record.error as { message?: unknown; code?: unknown };
    if (typeof error.message === 'string') {
      return error.message;
    }
    if (typeof error.code === 'string') {
      return error.code;
    }
  }

  if (typeof record.error === 'string') {
    return record.error;
  }

  return fallbackMessage;
}

async function requestJson<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, {
    credentials: 'same-origin',
    ...init,
  });
  const payload = await readJson(response);
  if (!response.ok) {
    throw new Error(responseErrorMessage(payload, response.statusText));
  }
  return payload as T;
}

export function getWorkspace(): Promise<WorkspaceStateResponse> {
  return requestJson<WorkspaceStateResponse>('/api/workspace');
}

export function uploadWorkspaceFile(file: File): Promise<WorkspaceUploadResponse> {
  const formData = new FormData();
  formData.append('midi_file', file);
  return requestJson<WorkspaceUploadResponse>('/api/workspace/uploads', {
    method: 'POST',
    body: formData,
  });
}

export async function deleteWorkspaceUpload(fileId: string): Promise<void> {
  const response = await fetch(`/api/workspace/uploads/${fileId}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    keepalive: true,
  });
  if (!response.ok && response.status !== 404) {
    const payload = await readJson(response);
    throw new Error(responseErrorMessage(payload, response.statusText));
  }
}

export function updateWorkspaceQueue(fileIds: string[]): Promise<WorkspaceQueueResponse> {
  return requestJson<WorkspaceQueueResponse>('/api/workspace/queue', {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ file_ids: fileIds }),
  });
}

export function saveWorkspaceConfig(config: WorkspaceConfigV1): Promise<WorkspaceConfigResponse> {
  return requestJson<WorkspaceConfigResponse>('/api/workspace/config', {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(config),
  });
}

export function createSynthesisJob(request: CreateSynthesisJobRequest): Promise<SynthesisJobResponse> {
  return requestJson<SynthesisJobResponse>('/api/synthesis-jobs', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });
}

export function getSynthesisJob(jobId: string): Promise<SynthesisJobResponse> {
  return requestJson<SynthesisJobResponse>(`/api/synthesis-jobs/${jobId}`);
}

export async function deleteSynthesisJob(jobIdOrDeleteUrl: string): Promise<void> {
  const response = await fetch(jobIdOrDeleteUrl.startsWith('/api/')
    ? jobIdOrDeleteUrl
    : `/api/synthesis-jobs/${jobIdOrDeleteUrl}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    keepalive: true,
  });
  if (!response.ok && response.status !== 404) {
    const payload = await readJson(response);
    throw new Error(responseErrorMessage(payload, response.statusText));
  }
}
