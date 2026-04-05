<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type ProxyLink } from '$lib/api';
	import { notificationStore } from '$lib/stores/notifications';
	import ProxyLinkView from '$lib/components/ProxyLink.svelte';

	let links = $state<ProxyLink[]>([]);
	let loading = $state(true);
	let label = $state('');

	async function load() {
		loading = true;
		try {
			const res = await api.links.list();
			links = res.links ?? [];
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to load links');
		} finally {
			loading = false;
		}
	}

	async function create() {
		try {
			await api.links.create(label || 'link');
			label = '';
			await load();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to create link');
		}
	}

	async function revoke(id: string) {
		try {
			await api.links.revoke(id);
			await load();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to revoke link');
		}
	}

	onMount(load);
</script>

<h1 class="text-2xl font-semibold mb-6">Links & Secrets</h1>

<div class="bg-gray-900 border border-gray-800 rounded-lg p-4 mb-4">
	<div class="text-sm text-gray-400 mb-3">Create new link</div>
	<div class="flex gap-2">
		<input bind:value={label} placeholder="Label (optional)" class="flex-1 bg-gray-950 border border-gray-700 rounded px-3 py-2" />
		<button class="px-3 py-2 rounded bg-cyan-700" onclick={create}>Generate</button>
	</div>
</div>

{#if loading}
	<p class="text-gray-400">Loading...</p>
{:else if links.length === 0}
	<p class="text-gray-500">No links created yet.</p>
{:else}
	<div class="space-y-3">
		{#each links as item}
			<div class="bg-gray-900 border border-gray-800 rounded-lg p-3">
				<div class="flex items-center justify-between gap-3 mb-2">
					<div class="text-xs text-gray-400">{item.label} · {item.active ? 'active' : 'revoked'}</div>
					{#if item.active}
						<button class="text-xs text-red-400 hover:text-red-300" onclick={() => revoke(item.id)}>Revoke</button>
					{/if}
				</div>
				<ProxyLinkView url={item.link} showQr />
			</div>
		{/each}
	</div>
{/if}
