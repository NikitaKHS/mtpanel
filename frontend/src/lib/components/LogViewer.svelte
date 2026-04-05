<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api';

	interface Props {
		autoRefresh?: boolean;
		refreshInterval?: number;
		lines?: number;
		level?: string;
	}

	let {
		autoRefresh = $bindable(false),
		refreshInterval = 5000,
		lines = $bindable(200),
		level = $bindable('all')
	}: Props = $props();

	let logLines = $state<string[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let logContainer = $state<HTMLElement | null>(null);
	let autoScrollEnabled = $state(true);
	let intervalHandle: ReturnType<typeof setInterval> | null = null;

	const levelOptions = ['all', 'debug', 'info', 'warn', 'error'];
	const lineOptions = [50, 100, 200, 500, 1000];

	const levelColors: Record<string, string> = {
		debug: 'text-gray-500',
		info: 'text-blue-400',
		warn: 'text-yellow-400',
		warning: 'text-yellow-400',
		error: 'text-red-400',
		fatal: 'text-red-500 font-bold'
	};

	function detectLineLevel(line: string): string {
		const lower = line.toLowerCase();
		if (/\berror\b|\bfatal\b/.test(lower)) return 'error';
		if (/\bwarn\b|\bwarning\b/.test(lower)) return 'warn';
		if (/\bdebug\b/.test(lower)) return 'debug';
		return 'info';
	}

	function lineClass(line: string): string {
		const lvl = detectLineLevel(line);
		return levelColors[lvl] ?? 'text-gray-300';
	}

	async function fetchLogs() {
		loading = true;
		error = null;
		try {
			const res = await api.proxy.logs(lines);
			logLines = res.lines;
			if (autoScrollEnabled) {
				setTimeout(scrollToBottom, 50);
			}
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Не удалось загрузить логи';
		} finally {
			loading = false;
		}
	}

	function scrollToBottom() {
		if (logContainer) {
			logContainer.scrollTop = logContainer.scrollHeight;
		}
	}

	function handleScroll() {
		if (!logContainer) return;
		const { scrollTop, scrollHeight, clientHeight } = logContainer;
		autoScrollEnabled = scrollHeight - scrollTop - clientHeight < 50;
	}

	function startAutoRefresh() {
		stopAutoRefresh();
		if (autoRefresh) {
			intervalHandle = setInterval(fetchLogs, refreshInterval);
		}
	}

	function stopAutoRefresh() {
		if (intervalHandle) {
			clearInterval(intervalHandle);
			intervalHandle = null;
		}
	}

	$effect(() => {
		if (autoRefresh) startAutoRefresh();
		else stopAutoRefresh();
	});

	$effect(() => {
		void level;
		void lines;
		fetchLogs();
	});

	onMount(() => {
		fetchLogs();
	});

	onDestroy(() => {
		stopAutoRefresh();
	});
</script>

<div class="flex flex-col h-full gap-3">
	<div class="flex flex-wrap items-center gap-3">
		<div class="flex items-center gap-2">
			<label class="text-xs text-gray-400" for="log-level">Уровень</label>
			<select id="log-level" bind:value={level} class="text-xs bg-gray-800 border border-gray-700 rounded px-2 py-1 text-gray-200 focus:outline-none focus:ring-1 focus:ring-cyan-500">
				{#each levelOptions as opt}
					<option value={opt}>{opt}</option>
				{/each}
			</select>
		</div>

		<div class="flex items-center gap-2">
			<label class="text-xs text-gray-400" for="log-lines">Строк</label>
			<select id="log-lines" bind:value={lines} class="text-xs bg-gray-800 border border-gray-700 rounded px-2 py-1 text-gray-200 focus:outline-none focus:ring-1 focus:ring-cyan-500">
				{#each lineOptions as opt}
					<option value={opt}>{opt}</option>
				{/each}
			</select>
		</div>

		<label class="flex items-center gap-2 cursor-pointer text-xs text-gray-400">
			<input type="checkbox" bind:checked={autoRefresh} class="rounded border-gray-600 bg-gray-800 text-cyan-500 focus:ring-cyan-500 focus:ring-offset-gray-900" />
			Автообновление
		</label>

		<button onclick={fetchLogs} disabled={loading} class="ml-auto flex items-center gap-1.5 text-xs px-2.5 py-1 rounded bg-gray-800 hover:bg-gray-700 text-gray-300 border border-gray-700 disabled:opacity-50 transition-colors">
			Обновить
		</button>

		<button onclick={scrollToBottom} class="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded bg-gray-800 hover:bg-gray-700 text-gray-300 border border-gray-700 transition-colors">
			Вниз
		</button>
	</div>

	{#if error}
		<div class="flex-1 flex items-center justify-center text-sm text-red-400">{error}</div>
	{:else}
		<div bind:this={logContainer} onscroll={handleScroll} class="flex-1 overflow-y-auto bg-gray-950 border border-gray-800 rounded-lg p-3 font-mono text-xs leading-5 min-h-0">
			{#if logLines.length === 0 && !loading}
				<p class="text-gray-600 text-center py-8">Записей логов пока нет</p>
			{:else}
				{#each logLines as line, i}
					<div class="hover:bg-gray-900/50 px-1 rounded {lineClass(line)}">
						<span class="select-none text-gray-700 mr-2">{String(i + 1).padStart(4, ' ')}</span>{line}
					</div>
				{/each}
			{/if}
		</div>
	{/if}

	<div class="text-xs text-gray-600">{logLines.length} строк</div>
</div>
