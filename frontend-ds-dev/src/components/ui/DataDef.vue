<script lang="ts" setup>
defineProps<{
	field: string
}>();
</script>
<template>
	<div class="px-4 sm:px-6 my-1 sm:grid sm:grid-cols-3 sm:gap-3 items-baseline">
		<dt class="font-medium sm:text-right">
			{{field}}
		</dt>
		<dd class="sm:col-span-2">
			<slot></slot>
		</dd>
	</div>
</template>
