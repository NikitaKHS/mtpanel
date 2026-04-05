<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type UpdateInfo } from '$lib/api';
	import { notificationStore } from '$lib/stores/notifications';

	let info = $state<UpdateInfo | null>(null);
	let loading = $state(true);
	let applying = $state(false);

	async function check() {
		loading = true;
		try {
			info = await api.updates.check();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to check updates');
		} finally {
			loading = false;
		}
	}

	async function apply() {
		applying = true;
		try {
			const res = await api.updates.apply();
			notificationStore.success(res.message || 'Update applied');
			await check();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Update failed');
		} finally {
			applying = false;
		}
	}

	onMount(check);
</script>

<h1 class="text-2xl font-semibold mb-6">Updates</h1>

{#if loading}
	<p class="text-gray-400">Checking for updates...</p>
{:else if info}
	<div class="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-2">
		<div class="text-sm text-gray-400">Current: <span class="text-gray-200">{info.current_version || 'unknown'}</span></div>
		<div class="text-sm text-gray-400">Latest: <span class="text-gray-200">{info.latest_version || 'unknown'}</span></div>
		<div class="text-sm">
			{#if info.update_available}
				<span class="text-yellow-400">Update available</span>
			{:else}
				<span class="text-emerald-400">Up to date</span>
			{/if}
		</div>

		<div class="flex gap-2 pt-2">
			<button class="px-3 py-2 rounded bg-gray-700" onclick={check}>Re-check</button>
			{#if info.update_available}
				<button disabled={applying} class="px-3 py-2 rounded bg-cyan-700 disabled:opacity-60" onclick={apply}>
					{applying ? 'Applying...' : 'Apply update'}
				</button>
			{/if}
		</div>
	</div>
{/if}
