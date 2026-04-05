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
			notificationStore.error('Passwords do not match');
			return;
		}
		loading = true;
		try {
			await authStore.setup(password);
			notificationStore.success('Setup completed');
			await goto('/dashboard');
		} catch (e) {
			const msg = e instanceof Error ? e.message : 'Setup failed';
			notificationStore.error(msg);
		} finally {
			loading = false;
		}
	}
</script>

<form onsubmit={submit} class="w-full max-w-md bg-gray-900 border border-gray-800 rounded-xl p-6">
	<h1 class="text-xl font-semibold mb-2">Initial Setup</h1>
	<p class="text-sm text-gray-400 mb-6">Create the first admin password (min 12 chars)</p>

	<label class="text-sm text-gray-300 block mb-2" for="password">New password</label>
	<input id="password" type="password" bind:value={password} required class="w-full bg-gray-950 border border-gray-700 rounded-lg px-3 py-2 mb-4" />

	<label class="text-sm text-gray-300 block mb-2" for="confirm">Confirm password</label>
	<input id="confirm" type="password" bind:value={confirm} required class="w-full bg-gray-950 border border-gray-700 rounded-lg px-3 py-2 mb-4" />

	<button type="submit" disabled={loading} class="w-full py-2 rounded-lg bg-cyan-600 hover:bg-cyan-700 disabled:opacity-60">
		{loading ? 'Saving...' : 'Complete setup'}
	</button>
</form>
