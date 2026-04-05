<script lang="ts">
	import LogViewer from '$lib/components/LogViewer.svelte';
</script>

<h1 class="text-2xl font-semibold mb-6">Логи</h1>
<div class="h-[70vh] rounded-2xl border border-slate-700 bg-slate-900/70 backdrop-blur p-3">
	<LogViewer autoRefresh={true} />
</div>
