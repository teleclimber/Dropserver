<script setup lang="ts">
import { computed } from 'vue';

import {app_control} from '../main';

const props = defineProps<{
	tab: string
}>();

const selected = computed( () => props.tab === app_control.tab );

</script>

<template>
	<a :href="tab" class="px-6 py-1 mr-2 uppercase font-bold text-sm" :class="{
			'bg-blue-50': !selected,
			'hover:bg-blue-100': !selected,
			'bg-black':selected,
			'text-white': selected
			}" @click.stop.prevent="app_control.setTab(tab)">
		<slot></slot>
	</a>
</template>