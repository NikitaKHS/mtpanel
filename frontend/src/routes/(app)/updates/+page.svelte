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
			notificationStore.error(e instanceof Error ? e.message : 'Не удалось проверить обновления');
		} finally {
			loading = false;
		}
	}

	async function apply() {
		applying = true;
		try {
			const res = await api.updates.apply();
			notificationStore.success(res.message || 'Обновление применено');
			await check();
		} catch (e) {
			notificationStore.error(e instanceof Error ? e.message : 'Ошибка обновления');
		} finally {
			applying = false;
		}
	}

	onMount(check);
</script>

<h1 class="text-2xl font-semibold mb-6">Обновления</h1>

{#if loading}
	<p class="text-slate-400">Проверяем обновления...</p>
{:else if info}
	<div class="rounded-2xl border border-slate-700 bg-slate-900/70 backdrop-blur p-5 space-y-2">
		<div class="text-sm text-slate-400">Текущая версия: <span class="text-slate-200">{info.current_version || 'unknown'}</span></div>
		<div class="text-sm text-slate-400">Последняя версия: <span class="text-slate-200">{info.latest_version || 'unknown'}</span></div>
		<div class="text-sm">
			{#if info.update_available}
				<span class="text-amber-400">Доступно обновление</span>
			{:else}
				<span class="text-emerald-400">Актуальная версия</span>
			{/if}
		</div>

		<div class="flex gap-2 pt-2">
			<button class="px-3 py-2 rounded-lg bg-slate-700 hover:bg-slate-600 transition-colors" onclick={check}>Проверить снова</button>
			{#if info.update_available}
				<button disabled={applying} class="px-3 py-2 rounded-lg bg-cyan-700 hover:bg-cyan-600 transition-colors disabled:opacity-60" onclick={apply}>
					{applying ? 'Применяем...' : 'Установить обновление'}
				</button>
			{/if}
		</div>
	</div>
{/if}
