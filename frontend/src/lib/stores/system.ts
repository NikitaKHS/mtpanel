import { writable } from 'svelte/store';
import { api } from '$lib/api';

interface SystemState {
	info: any | null;
	loading: boolean;
	error: string | null;
	cachedAt: Date | null;
}

const CACHE_TTL_MS = 30_000;

function createSystemStore() {
	const { subscribe, set, update } = writable<SystemState>({
		info: null,
		loading: false,
		error: null,
		cachedAt: null
	});

	async function fetchInfo(force = false): Promise<void> {
		// Return cached data if still fresh
		let cached: SystemState | null = null;
		subscribe((s) => { cached = s; })();
		if (!force && cached && (cached as SystemState).cachedAt) {
			const age = Date.now() - ((cached as SystemState).cachedAt as Date).getTime();
			if (age < CACHE_TTL_MS) return;
		}

		update((s) => ({ ...s, loading: true }));
		try {
			const info = await api.system.info();
			set({ info, loading: false, error: null, cachedAt: new Date() });
		} catch (e: unknown) {
			const msg = e instanceof Error ? e.message : 'Failed to fetch system info';
			update((s) => ({ ...s, loading: false, error: msg }));
		}
	}

	return { subscribe, fetchInfo };
}

export const systemStore = createSystemStore();
