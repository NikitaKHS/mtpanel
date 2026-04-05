<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiClientError } from '$lib/api';
	import { authStore } from '$lib/stores/auth';
	import { notificationStore } from '$lib/stores/notifications';

	let password = $state('');
	let loading = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		try {
			await authStore.login(password);
			await goto('/dashboard');
		} catch (e) {
			if (e instanceof ApiClientError && e.status === 428) {
				notificationStore.info('Первый запуск: сначала создайте пароль администратора.');
				await goto('/setup');
				return;
			}
			const msg = e instanceof Error ? e.message : 'Ошибка входа';
			notificationStore.error(msg);
		} finally {
			loading = false;
		}
	}
</script>

<form onsubmit={submit} class="w-full max-w-md rounded-2xl border border-cyan-500/20 bg-slate-900/85 backdrop-blur-xl p-7 shadow-2xl shadow-cyan-950/40">
	<p class="text-xs uppercase tracking-[0.16em] text-cyan-300/80 mb-2">MTPanel</p>
	<h1 class="text-2xl font-semibold mb-2 text-slate-50">Вход в панель</h1>
	<p class="text-sm text-slate-300 mb-6">Управление MTProxy на вашем сервере</p>

	<label class="text-sm text-slate-200 block mb-2" for="password">Пароль</label>
	<input
		id="password"
		type="password"
		bind:value={password}
		required
		class="w-full rounded-xl border border-slate-700 bg-slate-950/80 px-3 py-2.5 mb-4 text-slate-100 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
	/>

	<button
		type="submit"
		disabled={loading}
		class="w-full py-2.5 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-medium disabled:opacity-60 transition-colors"
	>
		{loading ? 'Выполняем вход...' : 'Войти'}
	</button>

	<a class="block text-center text-xs text-slate-400 mt-4 hover:text-cyan-300" href="/setup">Первый запуск / Setup</a>
</form>
