<template>
	<ViewWrap>
		<template v-if="appspace.loaded">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace</h3>
					<div class="flex items-stretch">
						<AppspaceStatusVisualizer :status="status" class="mr-4 flex items-center"></AppspaceStatusVisualizer>
						<button @click.stop.prevent="pause(!appspace.paused)" :disabled="pausing" class="btn btn-blue">
							{{ appspace.paused ? 'Unpause' : 'Pause'}}
						</button>
					</div>
				</div>
				<div class="px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
					<dt class="text-sm font-medium text-gray-500">Subdomain</dt>
					<dd class="mt-1 text-gray-900 sm:mt-0 sm:col-span-2">{{appspace.subdomain}}</dd>

					<dt class="text-sm font-medium text-gray-500">Created</dt>
					<dd class="mt-1 text-gray-900 sm:mt-0 sm:col-span-2">{{appspace.created_dt.toLocaleString()}}</dd>
	
					<dt class="text-sm font-medium text-gray-500">
						Application
					</dt>
					<dd class="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
						<dl class="border border-gray-200 rounded divide-y divide-gray-200">
							<DataDef field="App Name">{{app_version.app_name}}</DataDef>
							<DataDef field="Version">
								{{app_version.version}}
								<router-link v-if="appspace.upgrade" :to="{name: 'migrate-appspace', params:{id:appspace.id}, query:{to_version:appspace.upgrade.version}}" class="btn">
									<svg class="inline align-bottom w-6 h-6" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
										<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-8.707l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L9 9.414V13a1 1 0 102 0V9.414l1.293 1.293a1 1 0 001.414-1.414z" clip-rule="evenodd" />
									</svg>
									v{{appspace.upgrade.version}} available
								</router-link>
								<router-link v-else :to="{name: 'migrate-appspace', params:{id:appspace.id}}" class="btn">Show other versions</router-link>
							</DataDef>
							<DataDef field="Schema">{{app_version.schema}}</DataDef>
							<DataDef field="Dropserver API">{{app_version.api_version}}</DataDef>
						</dl>
						
					</dd>
				</div>
				
			</div>

			
		</template>
		<BigLoader v-else></BigLoader> 

	</ViewWrap>
</template>

<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, reactive, computed, onMounted, onUnmounted } from 'vue';

import { Appspace } from '../models/appspaces';
import { App } from '../models/apps';
import { AppVersion, AppVersionCollector } from '../models/app_versions';
import {setTitle} from '../controllers/nav';

import twineClient from '../twine-services/twine_client';
import { AppspaceStatus } from '../twine-services/appspace_status';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import AppspaceStatusVisualizer from '../components/AppspaceStatusVisualizer.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'ManageAppspace',
	components: {
		ViewWrap,
		BigLoader,
		AppspaceStatusVisualizer,
		DataDef
	},
	setup() {
		const route = useRoute();
		const appspace = reactive( new Appspace );
		const app_version = ref(new AppVersion);

		const status = reactive(new AppspaceStatus);
		
		const app = reactive(new App); 
		const show_all_versions = ref(false);
		function showAllVersions(show:boolean) {
			show_all_versions.value = show;
			if( show && !app.loaded ) {
				app.fetch(appspace.app_id);
			}
		}

		onMounted( async () => {
			const appspace_id = Number(route.params.id);
			await appspace.fetch(appspace_id);
			app_version.value = AppVersionCollector.get(appspace.app_id, appspace.app_version);

			setTitle(appspace.subdomain+".domain.sometld");
			
			// experimental
			status.connectStatus(appspace_id);
		});
		onUnmounted( async () => {
			status.disconnect();
			setTitle("");
		});

		const pausing = ref(false);

		async function pause(pause:boolean) {
			pausing.value = true;
			await appspace.setPause(pause);
			pausing.value = false;
		}

		return {
			appspace,
			app_version,
			status,
			show_all_versions,
			showAllVersions,
			app,
			pause,
			pausing,
		};
	}
});

</script>
