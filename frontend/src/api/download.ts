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
  const response = (error as { response?: { status?: number; data?: { code?: string }; headers?: Record<string, string> } })?.response;
  const code = response?.data?.code || response?.headers?.['x-zzz-error-code'];
  return (response?.status === 401 || response?.status === 403) && code !== 'auth_required';
}

function formatApiError(data: unknown, language: Locale): string {
  if (!data || typeof data !== 'object') return String(data || (language === 'zh' ? '请求失败' : 'Request failed'));
  const payload = data as { error?: unknown; hint?: unknown; code?: unknown; status?: number };
  const message = typeof payload.error === 'string' ? payload.error : language === 'zh' ? '请求失败' : 'Request failed';
  if (language === 'en') {
    const status = payload.status;
    const hint = payload.code === 'auth_required'
      ? 'A service access token is required. Enter the token configured by the administrator.'
      : payload.code === 'server_busy'
        ? 'The server is handling other downloads. Please wait and try again.'
        : payload.code === 'rate_limited'
          ? 'The zzz request rate limit has been reached. Please wait and try again.'
          : payload.code === 'github_rate_limit'
            ? 'The GitHub API rate limit has been reached. Try again later.'
            : payload.code === 'host_export_unavailable'
              ? 'The host export directory is not writable. Check the Docker mount permissions and ZZZ_UID/ZZZ_GID settings.'
              : status === 401
                ? 'The GitHub Token is invalid or expired. Check it and try again.'
                : status === 403
                  ? 'GitHub rejected the request. The API may be rate-limited or the repository may be inaccessible.'
                  : status === 502
                    ? 'The zzz server could not reach GitHub. Check the server network, DNS, and proxy settings.'
                    : status === 504
                      ? 'The GitHub request timed out. Try a narrower directory or increase GITHUB_TIMEOUT_SECS.'
                      : status === 404
                        ? 'The repository, branch, path, or selected files were not found.'
                        : status === 413
                          ? 'The selected content exceeds a server limit. Choose fewer or smaller files.'
                          : typeof payload.hint === 'string'
                            ? payload.hint
                            : '';
    return hint ? `${message}\n${hint}` : message;
  }
  return typeof payload.hint === 'string' ? `${message}\n${payload.hint}` : message;
}

export interface DownloadParams {
  url: string;
  ignore?: string[];
  include?: string[];
  select?: string[];
  selectAll?: boolean;
  token?: string;
  accessToken?: string;
}

export interface DownloadResponse extends AxiosResponse<ArrayBuffer> {
  filename?: string;
}

export interface Capabilities {
  host_export: boolean;
  auth_required: boolean;
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
  if (params.selectAll) searchParams.set('all', '1');

  const response = await axios.get<ArrayBuffer>(`/api/download?${searchParams.toString()}`, {
    responseType: 'arraybuffer',
    timeout: 300_000,
    headers: tokenHeaders(params.token, params.accessToken),
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
    all: Boolean(params.selectAll),
    destination: params.destination?.trim() || undefined,
    format: params.format,
  }, {
    timeout: 1_800_000,
    headers: tokenHeaders(params.token, params.accessToken),
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
  accessToken = '',
): Promise<TreeEntry[]> {
  const searchParams = new URLSearchParams({ url });
  if (ignore.length) searchParams.set('ignore', ignore.join(','));
  if (include.length) searchParams.set('include', include.join(','));
  const response = await axios.get<TreeEntry[]>(`/api/tree?${searchParams.toString()}`, {
    timeout: 300_000,
    headers: tokenHeaders(token, accessToken),
  });
  return response.data;
}

function tokenHeaders(githubToken?: string, accessToken?: string): Record<string, string> | undefined {
  const headers: Record<string, string> = {};
  const githubValue = githubToken?.trim();
  const accessValue = accessToken?.trim();
  if (githubValue) headers['X-GitHub-Token'] = githubValue;
  if (accessValue) headers['X-ZZZ-Access-Token'] = accessValue;
  return Object.keys(headers).length ? headers : undefined;
}
