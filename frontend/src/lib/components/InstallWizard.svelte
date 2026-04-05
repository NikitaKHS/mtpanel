<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { proxyStore } from '$lib/stores/proxy';
	import { notificationStore } from '$lib/stores/notifications';

	interface CompatibilityResult {
		compatible: boolean;
		os_ok: boolean;
		arch_ok: boolean;
		kernel_ok: boolean;
		warnings: string[];
		errors: string[];
	}

	interface Props {
		onInstalled?: () => void;
	}

	let { onInstalled }: Props = $props();

	type Step = 'check' | 'ready' | 'installing' | 'done' | 'error';

	let step = $state<Step>('check');
	let compat = $state<CompatibilityResult | null>(null);
	let compatLoading = $state(false);
	let installLoading = $state(false);
	let port = $state(443);
	let installMessage = $state('');
	let errorMsg = $state('');

	onMount(async () => {
		try {
			const settings = await api.settings.get();
			if (typeof settings?.proxy_port === 'number' && settings.proxy_port > 0) {
				port = settings.proxy_port;
			}
		} catch {
			// Keep default value if settings are unavailable.
		}
	});

	async function checkCompatibility() {
		compatLoading = true;
		try {
			const resp = await api.system.compatibility();
			compat = {
				compatible: !!resp.compatible,
				os_ok: true,
				arch_ok: true,
				kernel_ok: true,
				warnings: [],
				errors: resp.report?.issues ?? []
			};
			step = compat.compatible ? 'ready' : 'error';
			if (!compat.compatible) {
				errorMsg = compat.errors.join('; ') || 'Система не совместима';
			}
		} catch (e: unknown) {
			errorMsg = e instanceof Error ? e.message : 'Проверка совместимости не удалась';
			step = 'error';
		} finally {
			compatLoading = false;
		}
	}

	async function install() {
		installLoading = true;
		step = 'installing';
		try {
			const res = await api.proxy.install(port);
			installMessage = res.message;
			step = 'done';
			notificationStore.success('TeleMT успешно установлен');
			proxyStore.refresh();
			onInstalled?.();
		} catch (e: unknown) {
			errorMsg = e instanceof Error ? e.message : 'Ошибка установки';
			step = 'error';
			notificationStore.error(errorMsg);
		} finally {
			installLoading = false;
		}
	}

	const stepNums: Record<Step, number> = { check: 1, ready: 2, installing: 3, done: 4, error: 0 };
</script>

<div class="bg-gray-900 border border-gray-700 rounded-xl p-6 max-w-lg">
	<h2 class="text-base font-semibold text-gray-100 mb-1">Установка TeleMT</h2>
	<p class="text-sm text-gray-400 mb-6">Пройдите шаги, чтобы установить TeleMT на этот сервер.</p>

	<div class="flex items-center gap-2 mb-8 text-xs">
		{#each ['Проверка', 'Настройка', 'Установка', 'Готово'] as label, i}
			{@const num = i + 1}
			{@const current = stepNums[step]}
			<div class="flex items-center gap-2">
				<span class="w-6 h-6 rounded-full flex items-center justify-center font-semibold border
					{num < current ? 'bg-emerald-500 border-emerald-500 text-white' :
					 num === current ? 'bg-cyan-500 border-cyan-500 text-white' :
					 'border-gray-700 text-gray-600'}"
				>
					{#if num < current}
						<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
						</svg>
					{:else}
						{num}
					{/if}
				</span>
				<span class="{num === current ? 'text-gray-200' : 'text-gray-600'}">{label}</span>
			</div>
			{#if i < 3}
				<div class="flex-1 h-px {num < current ? 'bg-emerald-500/50' : 'bg-gray-800'}"></div>
			{/if}
		{/each}
	</div>

	{#if step === 'check'}
		<div class="text-center py-4">
			<p class="text-sm text-gray-400 mb-6">Проверьте совместимость системы перед установкой TeleMT.</p>
			<button
				onclick={checkCompatibility}
				disabled={compatLoading}
				class="px-5 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-700 text-white text-sm font-medium transition-colors disabled:opacity-50 flex items-center gap-2 mx-auto"
			>
				{#if compatLoading}
					<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
					</svg>
				{/if}
				Проверить совместимость
			</button>
		</div>

	{:else if step === 'ready'}
		<div class="space-y-4">
			{#if compat}
				<div class="grid grid-cols-2 gap-2 text-xs">
					{#each [['ОС', compat.os_ok], ['Архитектура', compat.arch_ok], ['Ядро', compat.kernel_ok]] as [label, ok]}
						<div class="flex items-center gap-2 bg-gray-800 rounded px-3 py-2">
							<span class="w-2 h-2 rounded-full {ok ? 'bg-emerald-400' : 'bg-red-400'}"></span>
							<span class="text-gray-400">{label}</span>
						</div>
					{/each}
				</div>
				{#if compat.warnings.length > 0}
					<div class="bg-yellow-500/10 border border-yellow-500/20 rounded p-3">
						<p class="text-xs font-medium text-yellow-400 mb-1">Предупреждения</p>
						{#each compat.warnings as w}
							<p class="text-xs text-yellow-300">{w}</p>
						{/each}
					</div>
				{/if}
			{/if}

			<div class="flex items-center gap-3">
				<label class="text-sm text-gray-400 shrink-0" for="install-port">Порт прокси</label>
				<input
					id="install-port"
					type="number"
					bind:value={port}
					min="1"
					max="65535"
					class="w-28 bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm text-gray-200 focus:outline-none focus:ring-1 focus:ring-cyan-500"
				/>
			</div>

			<button onclick={install} class="w-full py-2 rounded-lg bg-cyan-600 hover:bg-cyan-700 text-white text-sm font-medium transition-colors">
				Установить TeleMT
			</button>
		</div>

	{:else if step === 'installing'}
		<div class="text-center py-8">
			<svg class="w-10 h-10 animate-spin text-cyan-500 mx-auto mb-4" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
			</svg>
			<p class="text-sm text-gray-400">Устанавливаем TeleMT, пожалуйста подождите...</p>
		</div>

	{:else if step === 'done'}
		<div class="text-center py-6">
			<div class="w-12 h-12 rounded-full bg-emerald-500/20 flex items-center justify-center mx-auto mb-4">
				<svg class="w-6 h-6 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
			</div>
			<p class="text-sm font-medium text-gray-200 mb-2">Установка завершена</p>
			<p class="text-xs text-gray-500">{installMessage}</p>
		</div>

	{:else if step === 'error'}
		<div class="bg-red-500/10 border border-red-500/20 rounded-lg p-4">
			<p class="text-sm font-medium text-red-400 mb-1">Ошибка</p>
			<p class="text-xs text-red-300">{errorMsg}</p>
			<button onclick={() => { step = 'check'; errorMsg = ''; }} class="mt-3 text-xs text-red-400 hover:text-red-300 underline">Повторить</button>
		</div>
	{/if}
</div>
