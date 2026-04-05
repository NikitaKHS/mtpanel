<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { notificationStore } from '$lib/stores/notifications';

	let listenAddr = $state(':8080');
	let proxyPort = $state(443);
	let currentPassword = $state('');
	let newPassword = $state('');
	let loading = $state(true);

	async function load() {
		loading = true;
		try {
			const res = await api.settings.get();
			if (res.listen_addr) listenAddr = res.listen_addr;
			if (res.proxy_port) proxyPort = res.proxy_port;
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to load settings');
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		try {
			await api.settings.update({ listen_addr: listenAddr, proxy_port: proxyPort });
			notificationStore.success('Settings saved');
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to save settings');
		}
	}

	async function changePassword() {
		try {
			await api.settings.changePassword(currentPassword, newPassword);
			currentPassword = '';
			newPassword = '';
			notificationStore.success('Password changed');
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Failed to change password');
		}
	}

	onMount(load);
</script>

<h1 class="text-2xl font-semibold mb-6">Settings</h1>

{#if loading}
	<p class="text-gray-400">Loading...</p>
{:else}
	<div class="grid lg:grid-cols-2 gap-6">
		<div class="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-3">
			<h2 class="font-medium">Panel & Proxy</h2>
			<div>
				<label class="text-xs text-gray-400" for="listen-addr">Listen address</label>
				<input id="listen-addr" bind:value={listenAddr} class="w-full bg-gray-950 border border-gray-700 rounded px-3 py-2 mt-1" />
			</div>
			<div>
				<label class="text-xs text-gray-400" for="proxy-port">Proxy port</label>
				<input id="proxy-port" type="number" bind:value={proxyPort} class="w-full bg-gray-950 border border-gray-700 rounded px-3 py-2 mt-1" />
			</div>
			<button class="px-3 py-2 rounded bg-cyan-700" onclick={saveSettings}>Save settings</button>
		</div>

		<div class="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-3">
			<h2 class="font-medium">Change password</h2>
			<div>
				<label class="text-xs text-gray-400" for="current-password">Current password</label>
				<input id="current-password" type="password" bind:value={currentPassword} class="w-full bg-gray-950 border border-gray-700 rounded px-3 py-2 mt-1" />
			</div>
			<div>
				<label class="text-xs text-gray-400" for="new-password">New password</label>
				<input id="new-password" type="password" bind:value={newPassword} class="w-full bg-gray-950 border border-gray-700 rounded px-3 py-2 mt-1" />
			</div>
			<button class="px-3 py-2 rounded bg-cyan-700" onclick={changePassword}>Update password</button>
		</div>
	</div>
{/if}
