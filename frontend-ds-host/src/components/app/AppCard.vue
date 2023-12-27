<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { AppManifest } from '@/stores/types';

const props = defineProps<{
	manifest: AppManifest,
	icon_url: string
}>();

const icon_error = ref(false);
watch( () => props.icon_url, () => {
	icon_error.value = false;
});

const accent_color = computed( () => {
	if( props.manifest.accent_color ) return props.manifest.accent_color;
	return 'rgb(135, 151, 164)';
});

const release_date = computed( () => {
	if( !props.manifest.release_date ) return;
	return new Date(props.manifest.release_date).toLocaleDateString(undefined, {
		dateStyle:'medium'
	});
});

</script>
<template>
	<div class="my-8 px-4 sm:px-6">
		<div class=" mx-auto max-w-xl pb-4 bg-white shadow overflow-hidden border-2" style="border-top-width:1rem" :style="'border-color:'+accent_color" >
			<div class="grid app-grid gap-x-2 gap-y-2 px-2 py-2 ">
				<img v-if="icon_url && !icon_error" :src="icon_url" @error="icon_error = true" class="w-20 h-20" />
				<div v-else class="w-20 h-20 text-gray-300 flex justify-center items-center">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-14 h-14">
						<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
					</svg>
				</div>
				<div class="self-center">
					<h3 class="text-2xl leading-6 font-medium text-gray-900">{{manifest.name}}</h3>
					<p class="italic" v-if="manifest.short_description">“{{manifest.short_description}}”</p>
				</div>
				<div class="col-span-3 sm:col-start-2">
					<p class="text-lg font-medium">
						Version 
						<span class="bg-gray-200 text-gray-700 px-1 rounded-md">{{manifest.version}}</span>
						<span v-if="release_date"> released {{ release_date || '' }}</span>
					</p>
				</div>
			</div>
		</div>
	</div>
</template>