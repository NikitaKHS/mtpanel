import { writable, derived } from 'svelte/store';
import { api } from '$lib/api';

interface AuthState {
	token: string | null;
	expiresAt: Date | null;
	loading: boolean;
}

function createAuthStore() {
	// Hydrate from localStorage on init (browser only)
	const initial: AuthState = {
		token: null,
		expiresAt: null,
		loading: false
	};

	if (typeof localStorage !== 'undefined') {
		const stored = localStorage.getItem('mt_token');
		const exp = localStorage.getItem('mt_token_exp');
		if (stored) {
			initial.token = stored;
			initial.expiresAt = exp ? new Date(exp) : null;
		}
	}

	const { subscribe, set, update } = writable<AuthState>(initial);

	return {
		subscribe,

		async login(password: string): Promise<void> {
			update((s) => ({ ...s, loading: true }));
			try {
				const res = await api.auth.login(password);
				const expiresAt = new Date(res.expires_at);

				if (typeof localStorage !== 'undefined') {
					localStorage.setItem('mt_token', res.token);
					localStorage.setItem('mt_token_exp', res.expires_at);
				}

				set({ token: res.token, expiresAt, loading: false });
			} catch (e) {
				update((s) => ({ ...s, loading: false }));
				throw e;
			}
		},

		async setup(password: string): Promise<void> {
			update((s) => ({ ...s, loading: true }));
			try {
				const res = await api.auth.setup(password);
				const expiresAt = new Date(res.expires_at);
				if (typeof localStorage !== 'undefined') {
					localStorage.setItem('mt_token', res.token);
					localStorage.setItem('mt_token_exp', res.expires_at);
				}
				set({ token: res.token, expiresAt, loading: false });
			} catch (e) {
				update((s) => ({ ...s, loading: false }));
				throw e;
			}
		},

		async logout(): Promise<void> {
			try {
				await api.auth.logout();
			} catch {
				// best-effort
			} finally {
				if (typeof localStorage !== 'undefined') {
					localStorage.removeItem('mt_token');
					localStorage.removeItem('mt_token_exp');
				}
				set({ token: null, expiresAt: null, loading: false });
			}
		},

		clear() {
			if (typeof localStorage !== 'undefined') {
				localStorage.removeItem('mt_token');
				localStorage.removeItem('mt_token_exp');
			}
			set({ token: null, expiresAt: null, loading: false });
		}
	};
}

export const authStore = createAuthStore();

export const isAuthenticated = derived(authStore, ($auth) => {
	if (!$auth.token) return false;
	if ($auth.expiresAt && $auth.expiresAt < new Date()) return false;
	return true;
});
