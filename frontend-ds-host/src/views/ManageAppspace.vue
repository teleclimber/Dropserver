<template>
	<ViewWrap>
		<p>Subdomain: {{appspace.subdomain}}</p>
		<p>Created {{appspace.created_dt.toLocaleString()}}</p>
		<p>
			<span v-if="pausing">Pausing...</span>
			<button v-else-if="appspace.paused" @click.stop.prevent="unPause()">Un-Pause</button>
			<button v-else @click.stop.prevent="pause()">Pause</button>
		</p>
		<p v-if="status.loaded">Status Loaded. AppspaceID: {{status.appspace_id}}, Pause: {{ status.paused ? 'paused' : 'not paused'}}</p>

		<p>App Version: {{app_version.app_name}}, version: {{app_version.version}} (schema: {{app_version.schema}}, API: {{app_version.api_version}})</p>
		<p v-if="appspace.upgrade">
			Update available: {{appspace.upgrade.version}}
			<router-link :to="{name: 'migrate-appspace', params:{id:appspace.id}, query:{to_version:appspace.upgrade.version}}">Update</router-link>
		</p>
		<p v-else>
			No upgrade available
			<router-link :to="{name: 'migrate-appspace', params:{id:appspace.id}}">Show all versions</router-link>
		</p>
		

	</ViewWrap>
</template>

<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, reactive, onMounted, onUnmounted } from 'vue';

import { Appspace } from '../models/appspaces';
import { App } from '../models/apps';
import { AppVersion, AppVersionCollector } from '../models/app_versions';

import twineClient from '../twine-services/twine_client';
import { AppspaceStatus } from '../twine-services/appspace_status';

import ViewWrap from '../components/ViewWrap.vue';

export default defineComponent({
	name: 'ManageAppspace',
	components: {
		ViewWrap
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
			console.log("on mounted");
			const appspace_id = Number(route.params.id);
			await appspace.fetch(appspace_id);
			app_version.value = AppVersionCollector.get(appspace.app_id, appspace.app_version);
			
			// experimental
			status.connectStatus(appspace_id);
		});
		onUnmounted( async () => {
			status.disconnect();
		});

		const pausing = ref(false);

		function pause() {
			appspace.setPause(true);
		}
		function unPause() {
			appspace.setPause(false);
		}

		return {
			appspace,
			app_version,
			status,
			show_all_versions,
			showAllVersions,
			app,
			pause,
			unPause,
			pausing,
		};
	}
});

</script>
