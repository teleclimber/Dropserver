<template>
	<div class="md:mb-6 my-6 bg-yellow-100 shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-yellow-200">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Delete Appspace</h3>
			<p class="mt-1 max-w-2xl text-sm text-gray-700">
				Deleting this appspace will delete all the data (including backups!)
			</p>
		</div>
		<div class="pb-5">
			<DataDef field="Enter domain name of appspace:">
				<input type="text" v-model="domain_check" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
			</DataDef>
			<div class="px-4 sm:px-6 flex justify-end">
				<button v-if="!deleting" @click.stop.prevent="del" class="btn btn-blue" :disabled="!domain_checked">delete</button>
				<span v-else>Deleting...</span>
			</div>
		</div>
	</div>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, computed, onMounted, onUnmounted, PropType } from 'vue';
import router from '../../router/index';
import type {Appspace} from '../../models/appspaces';

import DataDef from '../../components/ui/DataDef.vue';

export default defineComponent({
	name: 'DeleteAppspace',
	components: {
		DataDef
	},
	props: {
		appspace: {
			type: Object as PropType<Appspace>,
			required: true
		}
	},
	setup(props) {

		const domain_check = ref("");

		const domain_checked = computed(() => {
			return domain_check.value.toLowerCase() === props.appspace.domain_name;
		});

		const deleting = ref(false);

		async function del() {
			if( !domain_checked.value) return;

			deleting.value = true;
			await props.appspace.del();

			router.push({name: 'appspaces'});
		}

		return {
			domain_check,
			domain_checked,
			deleting,
			del
		}
	}
});
</script>
