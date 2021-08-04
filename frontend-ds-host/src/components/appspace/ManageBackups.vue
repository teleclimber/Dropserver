<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Backups</h3>
			<div class="flex items-baseline">
				<router-link class="btn" :to="{name:'restore-appspace', params:{appspace_id:appspace_id}}">Restore</router-link>
				<button v-if="!backing_up_now" @click.stop.prevent="backupNow()" class="btn btn-blue ml-4">
					Backup Now
				</button>
				<span v-else>Backing up...</span>
			</div>
		</div>
		<div class="">
			<div v-for="file in appspaceBackups.files" :key="file.name" class="px-4 py-3 sm:px-6 border-b border-gray-200 flex items-baseline">
				<div class="flex-grow">{{file.name}}</div>
				<div>
					<button @click.stop.prevent="del(file.name)" class="btn text-red-700">
						<svg xmlns="http://www.w3.org/2000/svg" class="inline align-bottom h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
						</svg>
						<span class="hidden sm:inline-block">delete</span>
					</button>
				</div>
				<div class="px-4">[size]</div>
				<div>
					<a :href="file.download_link" class="btn">
						<svg xmlns="http://www.w3.org/2000/svg" class="inline align-bottom h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd" />
						</svg>
						<span class="hidden sm:inline-block">Download</span>
					</a>
				</div>
			</div>

		</div>
	</div>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, computed, onMounted, onUnmounted, PropType } from 'vue';

import {AppspaceBackups} from '../../models/appspace_backups';


export default defineComponent({
	name: 'ManageBackups',
	components: {
	},
	props: {
		appspace_id: {
			type: Number,
			required: true
		}
	},
	setup(props) {

		const appspaceBackups = reactive(new AppspaceBackups(props.appspace_id));
		appspaceBackups.fetchForAppspace();

		const backing_up_now = ref(false);
		async function backupNow() {
			backing_up_now.value = true;
			await appspaceBackups.backupNow();
			backing_up_now.value = false;
		}
		async function del(filename: string) {
			if( confirm("Delete "+filename+"?") ) {
				await appspaceBackups.delete(filename);
			}
		}
		return {
			appspaceBackups,
			backing_up_now,
			backupNow,
			del
		}
	}
});
</script>
