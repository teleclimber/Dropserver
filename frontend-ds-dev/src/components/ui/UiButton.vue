<script setup lang="ts">
defineProps<{
	pressed?: boolean
}>();

</script>
<template>
	<button class="	px-4 py-1 rounded border-t-2 border-b-2
			text-white font-medium text-xs leading-tight uppercase 
			focus:outline focus:outline-2 focus:outline-offset-2 focus:outline-cyan-600			
			active:bg-cyan-800
			disabled:bg-cyan-200 disabled:border-y-cyan-200 disabled:text-cyan-50
			transition duration-150 ease-in-out"
			:class="[ pressed ?  'bg-red-800 border-t-red-900 border-b-red-700 '
								+'hover:bg-orange-900 hover:border-t-black border-b-red-800' :
								 'bg-cyan-600  border-t-cyan-600 border-b-cyan-800 '
								+'hover:bg-cyan-700 hover:border-t-cyan-600 hover:border-b-cyan-900']">
		<slot></slot>
	</button>
</template>