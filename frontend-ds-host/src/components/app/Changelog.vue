<script setup lang="ts">
import SmallMessage from '@/components/ui/SmallMessage.vue';
defineProps<{
	changelog: string,
	error: string
}>();

</script>

<template>
	<div>
		<h4 class="px-4 sm:px-2 font-medium mt-6">What's changed:</h4>
		<div class="bg-gray-100 px-4 sm:px-2 py-2 max-h-48 overflow-y-scroll">
			<pre class="text-sm whitespace-pre-wrap">{{ changelog || "No changelog :(" }}</pre>
		</div>
		<SmallMessage mood="warn" class="mt-2" v-if="error">Problem loading changelog: {{ error }}</SmallMessage>
	</div>
</template>