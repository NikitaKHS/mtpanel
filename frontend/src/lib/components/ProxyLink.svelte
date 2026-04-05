<script lang="ts">
	import { notificationStore } from '$lib/stores/notifications';

	interface Props {
		url: string;
		label?: string;
		truncate?: boolean;
		showQr?: boolean;
	}

	let { url, label, truncate = true, showQr = false }: Props = $props();

	let copied = $state(false);
	let showQrModal = $state(false);

	async function copyToClipboard() {
		try {
			await navigator.clipboard.writeText(url);
			copied = true;
			notificationStore.success('Link copied to clipboard');
			setTimeout(() => (copied = false), 2000);
		} catch {
			notificationStore.error('Failed to copy to clipboard');
		}
	}

	function buildQrUrl(link: string) {
		return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(link)}`;
	}
</script>

<div class="flex items-center gap-2 min-w-0">
	<code
		class="flex-1 font-mono text-xs text-cyan-300 bg-gray-900 px-2 py-1.5 rounded border border-gray-700
		{truncate ? 'truncate' : 'break-all'}"
		title={url}
	>
		{label ? `[${label}] ` : ''}{url}
	</code>

	<button
		onclick={copyToClipboard}
		class="shrink-0 p-1.5 rounded hover:bg-gray-700 transition-colors text-gray-400 hover:text-white"
		title="Copy link"
	>
		{#if copied}
			<svg class="w-4 h-4 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
			</svg>
		{:else}
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
					d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
			</svg>
		{/if}
	</button>

	{#if showQr}
		<button
			onclick={() => (showQrModal = true)}
			class="shrink-0 p-1.5 rounded hover:bg-gray-700 transition-colors text-gray-400 hover:text-white"
			title="Show QR code"
		>
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
					d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
			</svg>
		</button>
	{/if}
</div>

{#if showQrModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
		role="presentation"
		onclick={() => (showQrModal = false)}
		onkeydown={(e) => e.key === 'Escape' && (showQrModal = false)}
	>
		<div class="bg-gray-900 border border-gray-700 rounded-xl p-6 flex flex-col items-center gap-4 max-w-xs w-full mx-4"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.key === 'Escape' && (showQrModal = false)}
			role="dialog"
			tabindex="0"
			aria-modal="true"
			aria-label="QR code"
		>
			<h3 class="text-sm font-semibold text-gray-200">Scan to connect</h3>
			<img
				src={buildQrUrl(url)}
				alt="QR code for proxy link"
				class="w-48 h-48 rounded bg-white p-2"
			/>
			<p class="text-xs text-gray-500 text-center break-all">{url}</p>
			<button
				onclick={() => (showQrModal = false)}
				class="text-xs text-gray-400 hover:text-white transition-colors"
			>Close</button>
		</div>
	</div>
{/if}
