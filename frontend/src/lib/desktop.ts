import { invoke } from '@tauri-apps/api/core';

type JsonMap = Record<string, unknown>;

let baseUrl = '';

export async function bootstrapDesktop() {
	if (baseUrl) {
		return { baseUrl };
	}
	const payload = await invoke<{ baseUrl: string }>('bootstrap');
	baseUrl = payload.baseUrl;
	return payload;
}

export async function rpc<T>(method: string, params: JsonMap = {}) {
	return invoke<T>('rpc', { method, params });
}

export async function pickDirectory(initial?: string) {
	return invoke<string | null>('pick_directory', {
		initial: initial || null
	});
}

export function getBaseUrl() {
	return baseUrl;
}

export function buildAssetUrl(kind: 'file' | 'render' | 'help', value: string) {
	if (!baseUrl) {
		throw new Error('desktop bootstrap has not completed');
	}
	return `${baseUrl}/${kind}/${encodeURIComponent(value)}`;
}
