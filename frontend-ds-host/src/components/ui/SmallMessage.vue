<script setup lang="ts">
import {computed} from 'vue';

const props = defineProps<{
	mood: 'info'|'warn'
}>();

const container_classes = computed( () => {
	if( props.mood === 'info' ) return ['border-sky-400','text-sky-700', 'bg-sky-50'];
	if( props.mood === 'warn' ) return ['border-orange-400','text-orange-700', 'bg-orange-50'];
});


</script>
<template>
	<div class="px-3 py-1 border rounded-lg" :class="container_classes">
		<svg v-if="mood === 'info'" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline align-text-bottom mr-1 text-sky-600">
			<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clip-rule="evenodd" />
		</svg>
		<svg v-if="mood === 'warn'" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline-block align-bottom mr-1 text-orange-500">
			<path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
		</svg>

		<slot></slot>
	</div>
</template>
