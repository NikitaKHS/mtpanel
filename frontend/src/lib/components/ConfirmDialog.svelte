<script lang="ts">
	interface Props {
		open: boolean;
		title: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		variant?: 'danger' | 'warning' | 'default';
		loading?: boolean;
		onconfirm: () => void;
		oncancel: () => void;
	}

	let {
		open,
		title,
		message,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		variant = 'default',
		loading = false,
		onconfirm,
		oncancel
	}: Props = $props();

	const confirmStyles = {
		danger: 'bg-red-600 hover:bg-red-700 text-white',
		warning: 'bg-yellow-600 hover:bg-yellow-700 text-white',
		default: 'bg-cyan-600 hover:bg-cyan-700 text-white'
	};

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') oncancel();
		if (e.key === 'Enter' && !loading) onconfirm();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
		role="presentation"
		onclick={oncancel}
		onkeydown={(e) => e.key === 'Escape' && oncancel()}
	>
		<div
			class="bg-gray-900 border border-gray-700 rounded-xl shadow-2xl p-6 max-w-sm w-full mx-4"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.key === 'Escape' && oncancel()}
			role="alertdialog"
			tabindex="0"
			aria-modal="true"
			aria-labelledby="dialog-title"
		>
			<h3 id="dialog-title" class="text-base font-semibold text-gray-100 mb-2">{title}</h3>
			<p class="text-sm text-gray-400 mb-6">{message}</p>

			<div class="flex justify-end gap-3">
				<button
					onclick={oncancel}
					disabled={loading}
					class="px-4 py-2 text-sm rounded-lg bg-gray-800 hover:bg-gray-700 text-gray-300 border border-gray-700 transition-colors disabled:opacity-50"
				>
					{cancelLabel}
				</button>
				<button
					onclick={onconfirm}
					disabled={loading}
					class="px-4 py-2 text-sm rounded-lg font-medium transition-colors disabled:opacity-50 flex items-center gap-2 {confirmStyles[variant]}"
				>
					{#if loading}
						<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
						</svg>
					{/if}
					{confirmLabel}
				</button>
			</div>
		</div>
	</div>
{/if}
