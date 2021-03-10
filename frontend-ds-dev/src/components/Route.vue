<template>
	<span v-if="route.auth.allow === 'public'" class="border-b border-gray-400 text-sm bold pl-2 bg-orange-300">public</span>
	<span v-else-if="route.auth.allow === 'authorized'"  class="border-b border-gray-400 text-sm bold pl-2 bg-teal-200">
		Auth: 
		<span v-if="route.auth.permission" class="text-white text-xs px-2 bg-teal-600">{{route.auth.permission}}</span>
		<span v-else>-</span>
	</span>
	<span v-else>???</span>

	<span class="border-b border-gray-400 pl-2" style="font-variant-caps: all-small-caps">{{route.methods.join(' ')}}</span>
	<span class="bg-gray-200 text-gray-900 pl-2 text-sm font-bold border-b border-gray-400">{{route.path}}</span>
	
	<template v-if="route.handler.type === 'file'">
		<span class="border-b border-gray-400 px-1 text-sm italic text-right">Serve files:</span>
		<span class="border-b border-gray-400 font-mono pl-2 ">{{route.handler.path}}</span>
	</template>
	<template v-else-if="route.handler.type === 'function'">
		<span class="border-b border-gray-400 px-1 text-sm italic text-right bg-orange-100">Call function:</span>
		<div class="border-b border-gray-400 bg-orange-100">
			<span class="font-mono pl-2">{{route.handler.file}}</span>
			<span class="italic font-mono pl-2 text-yellow-800">{{route.handler.function}}()</span>
		</div>
	</template>
	<template v-else>
		<span class="border-b border-gray-400 bg-yellow-500 text-yellow-100 px-1 text-sm">
			{{route.handler.type}}
		</span>
		<span class="border-b border-gray-400">not implemented</span>
	</template>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';
import type {RouteConfig} from '../models/appspace-routes-data';

export default defineComponent({
	name: 'AppspaceRoute',
	props: {
		route: {
			type: Object as PropType<RouteConfig>
		}
	}

});
</script>