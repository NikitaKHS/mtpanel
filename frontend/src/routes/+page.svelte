<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		const token = localStorage.getItem('mt_token');
		if (token) {
			goto('/dashboard');
			return;
		}
		goto('/login');
	});
</script>

<div class="min-h-screen flex items-center justify-center text-slate-400">Перенаправляем...</div>
