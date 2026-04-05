<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth';

	let { children } = $props();

	const nav = [
		{ href: '/dashboard', label: 'Dashboard' },
		{ href: '/proxy', label: 'Proxy' },
		{ href: '/links', label: 'Links' },
		{ href: '/logs', label: 'Logs' },
		{ href: '/updates', label: 'Updates' },
		{ href: '/settings', label: 'Settings' }
	];

	onMount(() => {
		const token = localStorage.getItem('mt_token');
		if (!token) goto('/login');
	});

	async function logout() {
		await authStore.logout();
		await goto('/login');
	}
</script>

<div class="min-h-screen bg-gray-950 text-gray-100">
	<header class="border-b border-gray-800">
		<div class="max-w-6xl mx-auto px-4 py-3 flex items-center gap-4">
			<div class="font-semibold">MTPanel</div>
			<nav class="flex items-center gap-3 text-sm">
				{#each nav as item}
					<a href={item.href} class="text-gray-300 hover:text-white">{item.label}</a>
				{/each}
			</nav>
			<button class="ml-auto text-sm text-gray-400 hover:text-white" onclick={logout}>Logout</button>
		</div>
	</header>

	<main class="max-w-6xl mx-auto px-4 py-6">
		{@render children()}
	</main>
</div>
