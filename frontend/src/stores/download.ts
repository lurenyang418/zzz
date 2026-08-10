import { writable } from 'svelte/store';

export type DownloadStatus = 'idle' | 'collecting' | 'done' | 'error';

export const downloadStatus = writable<DownloadStatus>('idle');
export const progress = writable(0);
export const fileCount = writable(0);
export const errorMessage = writable('');
