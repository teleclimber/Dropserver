<template>
	<ViewWrap>
		<p>Subdomain: {{appspace.subdomain}}</p>
		<p>Created {{appspace.created_dt.toLocaleString()}}</p>
		<p>App Version: {{app_version.app_name}}, version: {{appspace.app_version}} (schema: {{app_version.schema}}, API: {{app_version.api_version}})</p>
		<p>
			<span v-if="pausing">Pausing...</span>
			<button v-else-if="appspace.paused" @click.stop.prevent="unPause()">Un-Pause</button>
			<button v-else @click.stop.prevent="pause()">Pause</button>
		</p>
		<p v-if="status.loaded">Status Loaded. AppspaceID: {{status.appspace_id}}, Pause: {{ status.paused ? 'paused' : 'not paused'}}</p>
	</ViewWrap>
</template>

<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, reactive, onMounted, onUnmounted } from 'vue';

import { ReactiveAppspace, Appspace } from '../models/appspaces';
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
		const appspace = ReactiveAppspace();
		const app_version = ref(new AppVersion);

		const status = reactive(new AppspaceStatus);
		
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
			pause,
			unPause,
			pausing,
		};
	}
});

</script>
