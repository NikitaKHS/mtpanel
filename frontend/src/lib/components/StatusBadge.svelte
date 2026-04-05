<script lang="ts">
	import type { ProxyStatus } from '$lib/api';

	interface Props {
		status: ProxyStatus | 'unknown' | 'error';
		size?: 'sm' | 'md' | 'lg';
		showDot?: boolean;
	}

	let { status, size = 'md', showDot = true }: Props = $props();

	const labels: Record<string, string> = {
		running: 'Работает',
		stopped: 'Остановлен',
		error: 'Ошибка',
		failed: 'Сбой',
		unknown: 'Неизвестно'
	};

	const colors: Record<string, string> = {
		running: 'text-emerald-400 bg-emerald-400/10 ring-emerald-400/20',
		stopped: 'text-gray-400 bg-gray-400/10 ring-gray-400/20',
		error: 'text-red-400 bg-red-400/10 ring-red-400/20',
		failed: 'text-red-400 bg-red-400/10 ring-red-400/20',
		unknown: 'text-yellow-400 bg-yellow-400/10 ring-yellow-400/20'
	};

	const dotColors: Record<string, string> = {
		running: 'bg-emerald-400',
		stopped: 'bg-gray-500',
		error: 'bg-red-400',
		failed: 'bg-red-400',
		unknown: 'bg-yellow-400'
	};

	const sizes: Record<string, string> = {
		sm: 'text-xs px-1.5 py-0.5 gap-1',
		md: 'text-xs px-2 py-1 gap-1.5',
		lg: 'text-sm px-2.5 py-1 gap-2'
	};

	const dotSizes: Record<string, string> = {
		sm: 'w-1.5 h-1.5',
		md: 'w-2 h-2',
		lg: 'w-2.5 h-2.5'
	};
</script>

<span
	class="inline-flex items-center font-medium rounded ring-1 ring-inset {colors[status ?? 'unknown']} {sizes[size]}"
>
	{#if showDot}
		<span class="rounded-full shrink-0 {dotColors[status ?? 'unknown']} {dotSizes[size]}
			{status === 'running' ? 'animate-pulse' : ''}">
		</span>
	{/if}
	{labels[status ?? 'unknown'] ?? status}
</span>
