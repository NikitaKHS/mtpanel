export interface ApiError {
	message: string;
	status: number;
}

export type ProxyStatus = 'running' | 'stopped' | 'failed' | 'unknown';

export interface ProxyStatusResponse {
	status: ProxyStatus;
	port?: number;
	secret?: string;
}

export interface ProxyLink {
	id: string;
	label: string;
	secret: string;
	host: string;
	port: number;
	link: string;
	active: boolean;
	created_at: string;
	revoked_at?: string;
}

export interface UpdateInfo {
	current_version: string;
	latest_version: string;
	update_available: boolean;
	release_url: string;
	release_notes: string;
	published_at: string;
}

type Envelope<T> = {
	data?: T;
	error?: string;
};

const BASE = '/api';

class ApiClientError extends Error {
	status: number;
	constructor(message: string, status: number) {
		super(message);
		this.name = 'ApiClientError';
		this.status = status;
	}
}

function getToken(): string | null {
	if (typeof localStorage === 'undefined') return null;
	return localStorage.getItem('mt_token');
}

async function request<T>(method: string, path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
	const headers: Record<string, string> = { 'Content-Type': 'application/json' };
	const token = getToken();
	if (token) headers.Authorization = `Bearer ${token}`;

	const res = await fetch(`${BASE}${path}`, {
		method,
		headers,
		body: body === undefined ? undefined : JSON.stringify(body),
		signal
	});

	if (res.status === 401) {
		if (typeof localStorage !== 'undefined') {
			localStorage.removeItem('mt_token');
			localStorage.removeItem('mt_token_exp');
		}
		if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
			window.location.href = '/login';
		}
		throw new ApiClientError('Unauthorized', 401);
	}

	if (res.status === 204) {
		return undefined as T;
	}

	let payload: unknown = undefined;
	try {
		payload = await res.json();
	} catch {
		payload = undefined;
	}

	if (!res.ok) {
		const env = payload as Envelope<unknown>;
		const msg = env?.error || `HTTP ${res.status}`;
		throw new ApiClientError(msg, res.status);
	}

	const env = payload as Envelope<T>;
	if (env && typeof env === 'object' && 'data' in env) {
		return env.data as T;
	}
	return payload as T;
}

const get = <T>(path: string, signal?: AbortSignal) => request<T>('GET', path, undefined, signal);
const post = <T>(path: string, body?: unknown, signal?: AbortSignal) =>
	request<T>('POST', path, body, signal);
const put = <T>(path: string, body?: unknown) => request<T>('PUT', path, body);
const del = <T>(path: string) => request<T>('DELETE', path);

export const api = {
	auth: {
		login: (password: string) => post<{ token: string; expires_at: string }>('/auth/login', { password }),
		setup: (password: string) => post<{ token: string; expires_at: string }>('/auth/setup', { password }),
		logout: () => post<void>('/auth/logout'),
		changePassword: (currentPassword: string, newPassword: string) =>
			post<void>('/auth/change-password', {
				current_password: currentPassword,
				new_password: newPassword
			})
	},
	proxy: {
		status: (signal?: AbortSignal) => get<ProxyStatusResponse>('/proxy/status', signal),
		install: (port?: number) =>
			post<{ success: boolean; message: string }>('/proxy/install', port ? { port } : {}),
		start: () => post<void>('/proxy/start'),
		stop: () => post<void>('/proxy/stop'),
		restart: () => post<void>('/proxy/restart'),
		rotateSecret: () => post<{ secret: string }>('/proxy/rotate-secret'),
		setPort: (port: number) => post<void>('/proxy/port', { port }),
		logs: (lines = 200, signal?: AbortSignal) => get<{ lines: string[] }>(`/proxy/logs?lines=${lines}`, signal)
	},
	links: {
		list: (signal?: AbortSignal) => get<{ links: ProxyLink[] }>('/links', signal),
		create: (label: string) => post<ProxyLink>('/links', { label }),
		revoke: (id: string) => del<void>(`/links/${id}`)
	},
	system: {
		info: (signal?: AbortSignal) => get<any>('/system/info', signal),
		compatibility: (signal?: AbortSignal) => get<any>('/system/compatibility', signal)
	},
	updates: {
		check: (signal?: AbortSignal) => get<UpdateInfo>('/updates/check', signal),
		apply: () => post<{ success: boolean; message: string }>('/updates/apply')
	},
	audit: {
		list: (limit = 100, offset = 0, signal?: AbortSignal) =>
			get<{ events: any[]; total: number }>(`/audit?limit=${limit}&offset=${offset}`, signal)
	},
	settings: {
		get: (signal?: AbortSignal) => get<{ listen_addr?: string; proxy_port?: number }>('/settings', signal),
		update: (payload: { listen_addr?: string; proxy_port?: number }) => put<void>('/settings', payload),
		changePassword: (currentPassword: string, newPassword: string) =>
			post<void>('/settings/password', {
				current_password: currentPassword,
				new_password: newPassword
			})
	}
};

export { ApiClientError };
