<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { notificationStore } from '$lib/stores/notifications';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import InstallWizard from '$lib/components/InstallWizard.svelte';

	let status = $state<any>(null);
	let loading = $state(true);
	let port = $state(443);

	async function refresh() {
		loading = true;
		try {
			status = await api.proxy.status();
			if (status?.port) port = status.port;
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to load proxy status');
		} finally {
			loading = false;
		}
	}

	async function run(action: 'start' | 'stop' | 'restart') {
		try {
			if (action === 'start') await api.proxy.start();
			if (action === 'stop') await api.proxy.stop();
			if (action === 'restart') await api.proxy.restart();
			notificationStore.success(`Proxy ${action} successful`);
			await refresh();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : `Failed to ${action}`);
		}
	}

	async function rotateSecret() {
		try {
			await api.proxy.rotateSecret();
			notificationStore.success('Secret rotated');
			await refresh();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to rotate secret');
		}
	}

	async function updatePort() {
		try {
			await api.proxy.setPort(port);
			notificationStore.success('Port updated');
			await refresh();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to update port');
		}
	}

	onMount(refresh);
</script>

<h1 class="text-2xl font-semibold mb-6">Управление MTProxy</h1>

{#if loading}
	<p class="text-slate-400">Загрузка статуса...</p>
{:else}
	<div class="grid lg:grid-cols-2 gap-6">
		<div class="space-y-4">
			<div class="rounded-2xl border border-cyan-500/20 bg-slate-900/70 backdrop-blur p-5">
				<div class="flex items-center justify-between mb-3">
					<div class="text-sm text-slate-400">Текущий статус</div>
					<StatusBadge status={status?.status ?? 'unknown'} />
				</div>
				<div class="text-xs text-slate-500">Порт: {status?.port ?? 'не задан'}</div>
			</div>

			<div class="rounded-2xl border border-emerald-500/20 bg-slate-900/70 backdrop-blur p-5">
				<div class="text-sm text-slate-400 mb-3">Действия</div>
				<div class="flex gap-2">
					<button class="px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 transition-colors" onclick={() => run('start')}>Запустить</button>
					<button class="px-3 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 transition-colors" onclick={() => run('restart')}>Перезапустить</button>
					<button class="px-3 py-2 rounded-lg bg-rose-600 hover:bg-rose-500 transition-colors" onclick={() => run('stop')}>Остановить</button>
				</div>
				<button class="mt-3 px-3 py-2 rounded-lg bg-cyan-700 hover:bg-cyan-600 transition-colors" onclick={rotateSecret}>Сменить secret</button>
			</div>

			<div class="rounded-2xl border border-slate-700 bg-slate-900/70 backdrop-blur p-5">
				<div class="text-sm text-slate-400 mb-3">Порт</div>
				<div class="flex gap-2 items-center">
					<input type="number" min="1" max="65535" bind:value={port} class="bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 w-36" />
					<button class="px-3 py-2 rounded-lg bg-cyan-700 hover:bg-cyan-600 transition-colors" onclick={updatePort}>Применить</button>
				</div>
			</div>
		</div>

		<InstallWizard onInstalled={refresh} />
	</div>
{/if}
