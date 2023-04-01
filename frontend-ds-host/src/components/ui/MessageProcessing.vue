<script lang="ts" setup>
defineProps<{
	head: string
}>();
</script>

<template>
	<div class="bg-yellow-100 py-5 flex">
		<div class="w-12 sm:w-16 flex flex-shrink-0 justify-center">
			<svg class="w-8 h-8 text-yellow-600"  viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
				<path d="M4.75 11.75H8.25L10.25 4.75L13.75 19.25L15.75 11.75H19.25" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"></path>
			</svg>

		</div>
		<div class="pr-4 sm:pr-6">
			<h3 class="text-yellow-600 text-lg font-medium pb-2">{{head}}</h3>
			<slot></slot>
		</div>
	</div>
</template>
