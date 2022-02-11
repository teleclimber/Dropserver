<style scoped>
	.log-grid {
		display: grid;
		grid-template-columns: 12rem 10rem 1fr;
	}
</style>

<template>
	<div class="overflow-y-scroll bg-gray-100 h-full" ref="scroll_container">
		<div class="log-grid">
			<template  v-for="entry in live_log.entries" :key="entry.time">
				<span class="bg-gray-200 text-gray-800 pl-2 text-sm border-b border-gray-400">
					{{entry.time.toLocaleString()}}
				</span>
				<span class="bg-gray-200 text-gray-700 pl-2 text-sm font-bold border-b border-gray-400">{{entry.source}}</span>
				<pre class="px-2 border-b border-gray-400" :class="{'bg-red-200': entry.source.includes('stderr')}"
					>{{entry.message}}</pre>
			</template>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType, ref, Ref, watch, onMounted, nextTick } from 'vue';
import LiveLog from '../models/appspace-log-data';

export default defineComponent({
	name: 'Log',
	components: {
	},
	props: {
		title: {
			type: String,
			required: true
		},
		live_log: {
			type: LiveLog,
			required: true
		}
	},
	setup(props, context) {
		const scroll_container:Ref<undefined|HTMLElement> = ref(undefined);
		watch( () => props.live_log.entries, () => {
			nextTick( () => {
				if( !scroll_container.value ) return;
				scroll_container.value.scrollTop = scroll_container.value.scrollHeight;
				// make scrolling smooth after the initial scroll
				scroll_container.value.style.scrollBehavior = "smooth";
			});
		}, {deep: true} );
		return {
			scroll_container
		}
	},
});
</script>