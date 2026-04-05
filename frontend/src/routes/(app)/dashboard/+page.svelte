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

<h1 class="text-2xl font-semibold mb-6">Dashboard</h1>

{#if loading}
	<p class="text-gray-400">Loading...</p>
{:else}
	<div class="grid md:grid-cols-3 gap-4">
		<div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
			<div class="text-sm text-gray-400 mb-2">Proxy status</div>
			<StatusBadge status={status?.status ?? 'unknown'} />
		</div>
		<div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
			<div class="text-sm text-gray-400 mb-2">Proxy port</div>
			<div class="text-xl font-semibold">{status?.port ?? 'n/a'}</div>
		</div>
		<div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
			<div class="text-sm text-gray-400 mb-2">Host</div>
			<div class="text-xl font-semibold">{systemInfo?.hostname ?? 'n/a'}</div>
		</div>
	</div>
{/if}
