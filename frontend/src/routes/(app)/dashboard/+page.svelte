<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type ProxyStatusResponse } from '$lib/api';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let status: ProxyStatusResponse | null = $state(null);
	let systemInfo: any = $state(null);
	let loading = $state(true);

	async function load() {
		loading = true;
		try {
			[status, systemInfo] = await Promise.all([api.proxy.status(), api.system.info()]);
		} finally {
			loading = false;
		}
	}

	onMount(load);
</script>

<h1 class="text-2xl font-semibold mb-6">Обзор сервера</h1>

{#if loading}
	<p class="text-slate-400">Загрузка данных...</p>
{:else}
	<div class="grid md:grid-cols-3 gap-4">
		<div class="rounded-2xl border border-cyan-500/20 bg-slate-900/70 backdrop-blur p-5 shadow-lg shadow-cyan-950/20">
			<div class="text-sm text-slate-400 mb-2">Состояние TeleMT</div>
			<StatusBadge status={status?.status ?? 'unknown'} />
		</div>
		<div class="rounded-2xl border border-emerald-500/20 bg-slate-900/70 backdrop-blur p-5 shadow-lg shadow-emerald-950/20">
			<div class="text-sm text-slate-400 mb-2">Порт прокси</div>
			<div class="text-xl font-semibold">{status?.port ?? 'не задан'}</div>
		</div>
		<div class="rounded-2xl border border-slate-700 bg-slate-900/70 backdrop-blur p-5 shadow-lg shadow-black/20">
			<div class="text-sm text-slate-400 mb-2">Хост</div>
			<div class="text-xl font-semibold">{systemInfo?.hostname ?? 'не определен'}</div>
		</div>
	</div>
{/if}
