import axios, { type AxiosResponse } from 'axios';
import type { Locale } from '../i18n';

export function getApiError(error: unknown, language: Locale = 'zh'): string {
  const response = (error as { response?: { data?: unknown } })?.response;
  const data = response?.data;

  if (data instanceof ArrayBuffer) {
    const text = new TextDecoder().decode(data);
    try {
      return formatApiError(JSON.parse(text), language);
    } catch {
      return text || (language === 'zh' ? '请求失败' : 'Request failed');
    }
  }

  if (data && typeof data === 'object') return formatApiError(data, language);
  return error instanceof Error ? error.message : language === 'zh' ? '请求失败' : 'Request failed';
}

export function isGitHubAuthError(error: unknown): boolean {
  const status = (error as { response?: { status?: number } })?.response?.status;
  return status === 401 || status === 403;
}

function formatApiError(data: unknown, language: Locale): string {
  if (!data || typeof data !== 'object') return String(data || (language === 'zh' ? '请求失败' : 'Request failed'));
  const payload = data as { error?: unknown; hint?: unknown };
  const message = typeof payload.error === 'string' ? payload.error : language === 'zh' ? '请求失败' : 'Request failed';
  if (language === 'en' && typeof payload.hint === 'string') {
    const status = (data as { status?: number }).status;
    const hint = status === 401
      ? 'The GitHub Token is invalid or expired. Check it and try again.'
      : status === 403
        ? 'GitHub rejected the request. The API may be rate-limited or the repository may be inaccessible.'
        : status === 429
          ? 'The GitHub API rate limit has been reached. Try again later.'
          : payload.hint;
    return `${message}\n${hint}`;
  }
  return typeof payload.hint === 'string' ? `${message}\n${payload.hint}` : message;
}

export interface DownloadParams {
  url: string;
  ignore?: string[];
  include?: string[];
  select?: string[];
  token?: string;
}

export interface DownloadResponse extends AxiosResponse<ArrayBuffer> {
  filename?: string;
}

export interface Capabilities {
  host_export: boolean;
}

export interface HostExportParams extends DownloadParams {
  destination?: string;
  format: 'folder' | 'zip';
}

export interface HostExportResponse {
  format: 'folder' | 'zip';
  path: string;
  file_count: number;
  total_size: number;
}

export async function loadCapabilities(): Promise<Capabilities> {
  const response = await axios.get<Capabilities>('/api/capabilities', { timeout: 10_000 });
  return response.data;
}

export async function downloadFiles(params: DownloadParams): Promise<DownloadResponse> {
  const searchParams = new URLSearchParams({ url: params.url });
  if (params.ignore?.length) searchParams.set('ignore', params.ignore.join(','));
  if (params.include?.length) searchParams.set('include', params.include.join(','));
  if (params.select?.length) searchParams.set('select', params.select.join(','));

  const response = await axios.get<ArrayBuffer>(`/api/download?${searchParams.toString()}`, {
    responseType: 'arraybuffer',
    timeout: 300_000,
    headers: tokenHeaders(params.token),
  });
  const disposition = response.headers['content-disposition'] || '';
  const match = disposition.match(/filename="([^"]+)"/);
  return Object.assign(response, { filename: match?.[1] });
}

export async function exportToHost(params: HostExportParams): Promise<HostExportResponse> {
  const response = await axios.post<HostExportResponse>('/api/export', {
    url: params.url,
    ignore: params.ignore || [],
    include: params.include || [],
    select: params.select || [],
    destination: params.destination?.trim() || undefined,
    format: params.format,
  }, {
    timeout: 1_800_000,
    headers: tokenHeaders(params.token),
  });
  return response.data;
}

export interface TreeEntry {
  path: string;
  name: string;
  kind: 'file' | 'dir';
  size?: number;
}

export async function loadTree(
  url: string,
  ignore: string[] = [],
  include: string[] = [],
  token = '',
): Promise<TreeEntry[]> {
  const searchParams = new URLSearchParams({ url });
  if (ignore.length) searchParams.set('ignore', ignore.join(','));
  if (include.length) searchParams.set('include', include.join(','));
  const response = await axios.get<TreeEntry[]>(`/api/tree?${searchParams.toString()}`, {
    timeout: 300_000,
    headers: tokenHeaders(token),
  });
  return response.data;
}

function tokenHeaders(token?: string): Record<string, string> | undefined {
  const value = token?.trim();
  return value ? { 'X-GitHub-Token': value } : undefined;
}
