<script lang="ts" setup>

defineProps<{
	head: string
}>();

</script>

<template>
	<div class="bg-orange-100 py-5 flex">
		<div class="w-12 sm:w-16 flex flex-shrink-0 justify-center">
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-8 h-8 text-orange-500">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
			</svg>
		</div>
		<div class="pr-4 sm:pr-6">
			<h3 class="text-orange-600 text-lg font-medium pb-2">{{head}}</h3>
			<slot></slot>
		</div>
	</div>
</template>
