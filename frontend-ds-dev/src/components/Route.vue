<script setup lang="ts">
import {RouteConfig} from '../models/app-routes';

defineProps<{
	route: RouteConfig
}>();

</script>
<template>
	<span v-if="route.auth.allow === 'public'" class="border-b border-gray-400 text-sm bold pl-2 bg-yellow-300">public</span>
	<span v-else-if="route.auth.allow === 'authorized'"  class="border-b border-gray-400 text-sm bold pl-2 bg-teal-200">
		Auth: 
		<span v-if="route.auth.permission" class="text-white text-xs px-2 bg-teal-600">{{route.auth.permission}}</span>
		<span v-else>-</span>
	</span>
	<span v-else>???</span>

	<span class="border-b border-gray-400 pl-2" style="font-variant-caps: all-small-caps">{{route.method}}</span>
	<span class="bg-gray-200 text-gray-900 pl-2 text-sm font-bold border-b border-gray-400">
		{{route.path.path}}
		<span v-if="!route.path.end">**</span>
	</span>
	
	<template v-if="route.type === 'static'">
		<span class="border-b border-gray-400 px-1 text-sm italic text-right">Serve files:</span>
		<span class="border-b border-gray-400 font-mono pl-2 ">{{route.options.path}}</span>
	</template>
	<template v-else-if="route.type === 'function'">
		<span class="border-b border-gray-400 px-1 text-sm italic text-right bg-yellow-100">Call function:</span>
		<div class="border-b border-gray-400 bg-yellow-100">
			<span class="font-mono pl-2">{{route.options.name}}</span>
		</div>
	</template>
	<template v-else>
		<span class="border-b border-gray-400 bg-yellow-500 text-yellow-100 px-1 text-sm">
			{{route.type}}
		</span>
		<span class="border-b border-gray-400">not implemented</span>
	</template>
</template>