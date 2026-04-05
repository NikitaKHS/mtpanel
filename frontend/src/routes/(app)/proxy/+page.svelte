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

<h1 class="text-2xl font-semibold mb-6">Proxy Management</h1>

{#if loading}
	<p class="text-gray-400">Loading...</p>
{:else}
	<div class="grid lg:grid-cols-2 gap-6">
		<div class="space-y-4">
			<div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
				<div class="flex items-center justify-between mb-3">
					<div class="text-sm text-gray-400">Current status</div>
					<StatusBadge status={status?.status ?? 'unknown'} />
				</div>
				<div class="text-xs text-gray-500">Port: {status?.port ?? 'n/a'}</div>
			</div>

			<div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
				<div class="text-sm text-gray-400 mb-3">Actions</div>
				<div class="flex gap-2">
					<button class="px-3 py-2 rounded bg-emerald-600" onclick={() => run('start')}>Start</button>
					<button class="px-3 py-2 rounded bg-yellow-600" onclick={() => run('restart')}>Restart</button>
					<button class="px-3 py-2 rounded bg-red-600" onclick={() => run('stop')}>Stop</button>
				</div>
				<button class="mt-3 px-3 py-2 rounded bg-cyan-700" onclick={rotateSecret}>Rotate secret</button>
			</div>

			<div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
				<div class="text-sm text-gray-400 mb-3">Port</div>
				<div class="flex gap-2 items-center">
					<input type="number" min="1" max="65535" bind:value={port} class="bg-gray-950 border border-gray-700 rounded px-3 py-2 w-32" />
					<button class="px-3 py-2 rounded bg-cyan-700" onclick={updatePort}>Apply</button>
				</div>
			</div>
		</div>

		<InstallWizard onInstalled={refresh} />
	</div>
{/if}
