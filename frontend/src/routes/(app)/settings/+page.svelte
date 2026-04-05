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
			notificationStore.error(e instanceof Error ? e.message : 'Не удалось загрузить настройки');
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		try {
			await api.settings.update({ listen_addr: listenAddr, proxy_port: proxyPort });
			notificationStore.success('Настройки сохранены');
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Не удалось сохранить настройки');
		}
	}

	async function changePassword() {
		try {
			await api.settings.changePassword(currentPassword, newPassword);
			currentPassword = '';
			newPassword = '';
			notificationStore.success('Пароль обновлён');
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Не удалось сменить пароль');
		}
	}

	onMount(load);
</script>

<h1 class="text-2xl font-semibold mb-6">Настройки</h1>

{#if loading}
	<p class="text-slate-400">Загрузка...</p>
{:else}
	<div class="grid lg:grid-cols-2 gap-6">
		<div class="rounded-2xl border border-slate-700 bg-slate-900/70 backdrop-blur p-5 space-y-3">
			<h2 class="font-medium">Панель и прокси</h2>
			<div>
				<label class="text-xs text-slate-400" for="listen-addr">Адрес панели</label>
				<input id="listen-addr" bind:value={listenAddr} class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 mt-1" />
			</div>
			<div>
				<label class="text-xs text-slate-400" for="proxy-port">Порт MTProxy</label>
				<input id="proxy-port" type="number" bind:value={proxyPort} class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 mt-1" />
			</div>
			<button class="px-3 py-2 rounded-lg bg-cyan-700 hover:bg-cyan-600 transition-colors" onclick={saveSettings}>Сохранить</button>
		</div>

		<div class="rounded-2xl border border-slate-700 bg-slate-900/70 backdrop-blur p-5 space-y-3">
			<h2 class="font-medium">Смена пароля</h2>
			<div>
				<label class="text-xs text-slate-400" for="current-password">Текущий пароль</label>
				<input id="current-password" type="password" bind:value={currentPassword} class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 mt-1" />
			</div>
			<div>
				<label class="text-xs text-slate-400" for="new-password">Новый пароль</label>
				<input id="new-password" type="password" bind:value={newPassword} class="w-full bg-slate-950 border border-slate-700 rounded-lg px-3 py-2 mt-1" />
			</div>
			<button class="px-3 py-2 rounded-lg bg-cyan-700 hover:bg-cyan-600 transition-colors" onclick={changePassword}>Обновить пароль</button>
		</div>
	</div>
{/if}
