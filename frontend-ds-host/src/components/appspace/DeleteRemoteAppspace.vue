<template>
	<div class="md:mb-6 my-6 bg-yellow-100 shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-yellow-200">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Remove Remote Appspace</h3>
			<p class="mt-1 max-w-2xl text-sm text-gray-700">
				Removing this remote appspace means you will no longer be able to access it
			</p>
		</div>
		<div class="py-5">
			<div class="px-4 sm:px-6 flex justify-end">
				<button v-if="!deleting" @click.stop.prevent="del" class="btn btn-blue" >remove</button>
				<span v-else>Removing...</span>
			</div>
		</div>
	</div>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, computed, onMounted, onUnmounted, PropType } from 'vue';
import router from '../../router/index';
import type {RemoteAppspace} from '../../models/remote_appspaces';

import DataDef from '../../components/ui/DataDef.vue';

export default defineComponent({
	name: 'DeleteRemoteAppspace',
	components: {
		DataDef
	},
	props: {
		appspace: {
			type: Object as PropType<RemoteAppspace>,
			required: true
		}
	},
	setup(props) {
		const deleting = ref(false);

		async function del() {
			deleting.value = true;
			await props.appspace.del();

			router.push({name: 'appspaces'});
		}

		return {
			deleting,
			del
		}
	}
});
</script>
