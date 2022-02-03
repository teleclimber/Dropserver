<template>
	<button class="bg-cyan-600 
			px-4 py-1 rounded 
			border-t-2 border-t-cyan-600
			border-b-2 border-b-cyan-800
			text-white font-medium text-xs leading-tight uppercase 
			hover:bg-cyan-700 hover:border-t-cyan-600 hover:border-b-cyan-900
			focus:outline focus:outline-2 focus:outline-offset-2 focus:outline-cyan-600			
			active:bg-cyan-800
			disabled:bg-cyan-200 disabled:border-y-cyan-200 disabled:text-cyan-50
			transition duration-150 ease-in-out"
			:class="{'bg-cyan-800':pressed, 'border-t-cyan-900':pressed, 'border-b-cyan-600':pressed}">
		<slot></slot>
	</button>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
	components: {
	},
	props: {
		pressed: {
			type: Boolean,
			required: false,
			default: false
		}
	},
	setup(props, context) {

	}
});
</script>