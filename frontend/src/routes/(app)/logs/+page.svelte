<script lang="ts">
	import LogViewer from '$lib/components/LogViewer.svelte';
</script>

<h1 class="text-2xl font-semibold mb-6">Logs</h1>
<div class="h-[70vh]">
	<LogViewer autoRefresh={true} />
</div>
