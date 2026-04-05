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
			notificationStore.error(e instanceof Error ? e.message : 'Не удалось загрузить ссылки');
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
			notificationStore.error(e instanceof Error ? e.message : 'Не удалось создать ссылку');
		}
	}

	async function revoke(id: string) {
		try {
			await api.links.revoke(id);
			await load();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Не удалось отозвать ссылку');
		}
	}

	onMount(load);
</script>

<h1 class="text-2xl font-semibold mb-6">Ссылки и секреты</h1>

<div class="rounded-2xl border border-cyan-500/20 bg-slate-900/70 backdrop-blur p-5 mb-4">
	<div class="text-sm text-slate-400 mb-3">Создать новую ссылку</div>
	<div class="flex gap-2">
		<input bind:value={label} placeholder="Название (опционально)" class="flex-1 bg-slate-950 border border-slate-700 rounded-lg px-3 py-2" />
		<button class="px-3 py-2 rounded-lg bg-cyan-700 hover:bg-cyan-600 transition-colors" onclick={create}>Сгенерировать</button>
	</div>
</div>

{#if loading}
	<p class="text-slate-400">Загрузка ссылок...</p>
{:else if links.length === 0}
	<p class="text-slate-500">Ссылок пока нет.</p>
{:else}
	<div class="space-y-3">
		{#each links as item}
			<div class="rounded-2xl border border-slate-700 bg-slate-900/70 backdrop-blur p-4">
				<div class="flex items-center justify-between gap-3 mb-2">
					<div class="text-xs text-slate-400">{item.label} · {item.active ? 'активна' : 'отозвана'}</div>
					{#if item.active}
						<button class="text-xs text-rose-400 hover:text-rose-300" onclick={() => revoke(item.id)}>Отозвать</button>
					{/if}
				</div>
				<ProxyLinkView url={item.link} showQr />
			</div>
		{/each}
	</div>
{/if}
