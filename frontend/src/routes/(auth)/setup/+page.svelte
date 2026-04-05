<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth';
	import { notificationStore } from '$lib/stores/notifications';

	let password = $state('');
	let confirm = $state('');
	let loading = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (password !== confirm) {
			notificationStore.error('Пароли не совпадают');
			return;
		}
		loading = true;
		try {
			await authStore.setup(password);
			notificationStore.success('Первичная настройка завершена');
			await goto('/dashboard');
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Ошибка первичной настройки';
			notificationStore.error(msg);
		} finally {
			loading = false;
		}
	}
</script>

<form onsubmit={submit} class="w-full max-w-md rounded-2xl border border-emerald-500/20 bg-slate-900/85 backdrop-blur-xl p-7 shadow-2xl shadow-emerald-950/40">
	<p class="text-xs uppercase tracking-[0.16em] text-emerald-300/80 mb-2">First Run</p>
	<h1 class="text-2xl font-semibold mb-2 text-slate-50">Первичная настройка</h1>
	<p class="text-sm text-slate-300 mb-6">Создайте пароль администратора (минимум 12 символов)</p>

	<label class="text-sm text-slate-200 block mb-2" for="password">Новый пароль</label>
	<input id="password" type="password" bind:value={password} required class="w-full rounded-xl border border-slate-700 bg-slate-950/80 px-3 py-2.5 mb-4 text-slate-100 focus:outline-none focus:ring-2 focus:ring-emerald-500/50" />

	<label class="text-sm text-slate-200 block mb-2" for="confirm">Повторите пароль</label>
	<input id="confirm" type="password" bind:value={confirm} required class="w-full rounded-xl border border-slate-700 bg-slate-950/80 px-3 py-2.5 mb-4 text-slate-100 focus:outline-none focus:ring-2 focus:ring-emerald-500/50" />

	<button type="submit" disabled={loading} class="w-full py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-medium disabled:opacity-60 transition-colors">
		{loading ? 'Сохраняем...' : 'Завершить настройку'}
	</button>
</form>
