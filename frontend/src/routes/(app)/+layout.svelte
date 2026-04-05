<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth';

	let { children } = $props();

	const nav = [
		{ href: '/dashboard', label: 'Обзор' },
		{ href: '/proxy', label: 'Прокси' },
		{ href: '/links', label: 'Ссылки' },
		{ href: '/logs', label: 'Логи' },
		{ href: '/updates', label: 'Обновления' },
		{ href: '/settings', label: 'Настройки' }
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

<div class="min-h-screen bg-slate-950 text-slate-100">
	<div class="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(34,211,238,0.13),transparent_38%),radial-gradient(circle_at_80%_80%,rgba(16,185,129,0.1),transparent_35%)]"></div>
	<header class="sticky top-0 z-20 border-b border-slate-800/70 bg-slate-950/80 backdrop-blur-xl">
		<div class="max-w-6xl mx-auto px-4 py-3 flex items-center gap-4">
			<div class="font-semibold tracking-wide text-cyan-300">MTPanel</div>
			<nav class="flex items-center gap-2 text-sm">
				{#each nav as item}
					<a
						href={item.href}
						class="px-3 py-1.5 rounded-lg border transition-colors {page.url.pathname === item.href ? 'bg-cyan-500/15 border-cyan-400/40 text-cyan-200' : 'border-transparent text-slate-300 hover:border-slate-700 hover:text-white'}"
					>{item.label}</a>
				{/each}
			</nav>
			<button class="ml-auto text-sm rounded-lg px-3 py-1.5 border border-slate-700 text-slate-300 hover:text-white hover:border-slate-500 transition-colors" onclick={logout}>Выйти</button>
		</div>
	</header>

	<main class="relative max-w-6xl mx-auto px-4 py-6">
		{@render children()}
	</main>
</div>
