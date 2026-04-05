<script lang="ts">
	import { goto } from '$app/navigation';
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
			const msg = e instanceof Error ? e.message : 'Login failed';
			notificationStore.error(msg);
		} finally {
			loading = false;
		}
	}
</script>

<form onsubmit={submit} class="w-full max-w-md bg-gray-900 border border-gray-800 rounded-xl p-6">
	<h1 class="text-xl font-semibold mb-2">MTPanel Login</h1>
	<p class="text-sm text-gray-400 mb-6">Sign in to manage MTProxy</p>

	<label class="text-sm text-gray-300 block mb-2" for="password">Password</label>
	<input
		id="password"
		type="password"
		bind:value={password}
		required
		class="w-full bg-gray-950 border border-gray-700 rounded-lg px-3 py-2 mb-4"
	/>

	<button
		type="submit"
		disabled={loading}
		class="w-full py-2 rounded-lg bg-cyan-600 hover:bg-cyan-700 disabled:opacity-60"
	>
		{loading ? 'Signing in...' : 'Sign in'}
	</button>

	<a class="block text-center text-xs text-gray-500 mt-4 hover:text-gray-300" href="/setup">First run setup</a>
</form>
