import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id: string;
	type: ToastType;
	message: string;
	duration: number;
}

function createNotificationStore() {
	const { subscribe, update } = writable<Toast[]>([]);

	function makeId(): string {
		if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
			return crypto.randomUUID();
		}
		// Fallback for insecure contexts/older runtimes.
		return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
	}

	function add(type: ToastType, message: string, duration = 4000): void {
		const id = makeId();
		update((toasts) => [...toasts, { id, type, message, duration }]);

		if (duration > 0) {
			setTimeout(() => remove(id), duration);
		}
	}

	function remove(id: string): void {
		update((toasts) => toasts.filter((t) => t.id !== id));
	}

	return {
		subscribe,
		success: (msg: string, duration?: number) => add('success', msg, duration),
		error: (msg: string, duration?: number) => add('error', msg, duration ?? 6000),
		warning: (msg: string, duration?: number) => add('warning', msg, duration),
		info: (msg: string, duration?: number) => add('info', msg, duration),
		remove
	};
}

export const notificationStore = createNotificationStore();
