<script lang="ts">
	let { children } = $props();
</script>

<div class="relative min-h-screen overflow-hidden bg-slate-950 text-slate-100 flex items-center justify-center p-4">
	<div class="pointer-events-none absolute -top-24 -left-24 h-72 w-72 rounded-full bg-cyan-500/20 blur-3xl"></div>
	<div class="pointer-events-none absolute -bottom-24 -right-24 h-72 w-72 rounded-full bg-emerald-500/20 blur-3xl"></div>
	<div class="relative w-full flex items-center justify-center">
		{@render children()}
	</div>
</div>
