import { writable, derived } from 'svelte/store';
import { api, type ProxyStatusResponse } from '$lib/api';
import { notificationStore } from './notifications';

interface ProxyState {
	status: ProxyStatusResponse | null;
	loading: boolean;
	error: string | null;
	lastUpdated: Date | null;
	actionLoading: string | null; // 'start' | 'stop' | 'restart' | null
}

function createProxyStore() {
	const { subscribe, set, update } = writable<ProxyState>({
		status: null,
		loading: false,
		error: null,
		lastUpdated: null,
		actionLoading: null
	});

	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let pollController: AbortController | null = null;

	async function fetchStatus(): Promise<void> {
		pollController?.abort();
		pollController = new AbortController();

		try {
			const status = await api.proxy.status(pollController.signal);
			update((s) => ({
				...s,
				status,
				error: null,
				loading: false,
				lastUpdated: new Date()
			}));
		} catch (e: unknown) {
			if (e instanceof Error && e.name === 'AbortError') return;
			const msg = e instanceof Error ? e.message : 'Failed to fetch status';
			update((s) => ({ ...s, error: msg, loading: false }));
		}
	}

	function startPolling(intervalMs = 5000): void {
		stopPolling();
		fetchStatus();

		const handleVisibility = () => {
			if (document.hidden) {
				stopPolling();
			} else {
				fetchStatus();
				pollInterval = setInterval(fetchStatus, intervalMs);
			}
		};

		pollInterval = setInterval(fetchStatus, intervalMs);

		if (typeof document !== 'undefined') {
			document.addEventListener('visibilitychange', handleVisibility);
		}
	}

	function stopPolling(): void {
		if (pollInterval) {
			clearInterval(pollInterval);
			pollInterval = null;
		}
		pollController?.abort();
	}

	async function runAction(action: 'start' | 'stop' | 'restart'): Promise<void> {
		update((s) => ({ ...s, actionLoading: action }));
		try {
			if (action === 'start') await api.proxy.start();
			else if (action === 'stop') await api.proxy.stop();
			else if (action === 'restart') await api.proxy.restart();

			notificationStore.success(`Proxy ${action}ed successfully`);
			await fetchStatus();
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : `Failed to ${action} proxy`;
			notificationStore.error(msg);
		} finally {
			update((s) => ({ ...s, actionLoading: null }));
		}
	}

	async function rotateSecret(): Promise<string | null> {
		try {
			const res = await api.proxy.rotateSecret();
			notificationStore.success('Secret rotated');
			await fetchStatus();
			return res.secret;
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : 'Failed to rotate secret';
			notificationStore.error(msg);
			return null;
		}
	}

	return {
		subscribe,
		fetchStatus,
		startPolling,
		stopPolling,
		runAction,
		rotateSecret,
		refresh: fetchStatus
	};
}

export const proxyStore = createProxyStore();

export const proxyRunning = derived(
	proxyStore,
	($proxy) => $proxy.status?.status === 'running'
);
