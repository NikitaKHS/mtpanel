<script lang="ts">
	let { children } = $props();
</script>

<div class="min-h-screen bg-gray-950 text-gray-100 flex items-center justify-center p-4">
	{@render children()}
</div>
