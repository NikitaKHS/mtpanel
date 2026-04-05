<script lang="ts">
	import { notificationStore, type Toast } from '$lib/stores/notifications';
	import { fly, fade } from 'svelte/transition';
	import { flip } from 'svelte/animate';

	const icons: Record<string, string> = {
		success: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
		error: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z',
		warning: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z',
		info: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z'
	};

	const colors: Record<string, string> = {
		success: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
		error: 'border-red-500/30 bg-red-500/10 text-red-300',
		warning: 'border-yellow-500/30 bg-yellow-500/10 text-yellow-300',
		info: 'border-blue-500/30 bg-blue-500/10 text-blue-300'
	};

	const iconColors: Record<string, string> = {
		success: 'text-emerald-400',
		error: 'text-red-400',
		warning: 'text-yellow-400',
		info: 'text-blue-400'
	};
</script>

<div class="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 max-w-sm w-full pointer-events-none px-4">
	{#each $notificationStore as toast (toast.id)}
		<div
			animate:flip={{ duration: 200 }}
			in:fly={{ y: 20, duration: 250 }}
			out:fade={{ duration: 150 }}
			class="pointer-events-auto flex items-start gap-3 px-4 py-3 rounded-lg border backdrop-blur-sm shadow-lg {colors[toast.type]}"
		>
			<svg class="w-5 h-5 shrink-0 mt-0.5 {iconColors[toast.type]}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d={icons[toast.type]} />
			</svg>
			<p class="flex-1 text-sm leading-relaxed">{toast.message}</p>
			<button
				onclick={() => notificationStore.remove(toast.id)}
				class="shrink-0 opacity-60 hover:opacity-100 transition-opacity"
				aria-label="Dismiss"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/each}
</div>
