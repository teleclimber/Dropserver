<template>
	<div class="bg-green-100 py-5 flex">
		<div class="w-12 sm:w-16 flex flex-shrink-0 justify-center">
			<svg class="w-8 h-8 text-green-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
		</div>
		<div class="pr-4 sm:pr-6">
			<h3 class="text-green-600 text-lg font-medium pb-2">{{head}}</h3>
			<slot></slot>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'MessageHappy',
  components: {
    
  },
  props: ['head']
});
</script>