<template>
	<ViewWrap>
		<h3>Manage appspace</h3>

		<p>Subdomain: {{appspace.subdomain}}</p>
		<p>Created {{appspace.created_dt.toLocaleString()}}</p>
		<p>App Version: {{app_version.app_name}}, version: {{appspace.app_version}} (schema: {{app_version.schema}}, API: {{app_version.api_version}}</p>
		<p>
			<span v-if="pausing">Pausing...</span>
			<button v-else-if="appspace.paused" @click.stop.prevent="unPause()">Un-Pause</button>
			<button v-else @click.stop.prevent="pause()">Pause</button>
		</p>
	</ViewWrap>
</template>

<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, reactive, onMounted } from 'vue';

import { ReactiveAppspace, Appspace } from '../models/appspaces';
import { AppVersion, AppVersionCollector } from '../models/app_versions';

import ViewWrap from '../components/ViewWrap.vue';

export default defineComponent({
	name: 'ManageAppspace',
	components: {
		ViewWrap
	},
	setup() {
		const route = useRoute();
		const appspace = ReactiveAppspace();
		const app_version = ref(new AppVersion)
		onMounted( async () => {
			await appspace.fetch(Number(route.params.id));
			app_version.value = AppVersionCollector.get(appspace.app_id, appspace.app_version);
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
			pause,
			unPause,
			pausing,
		};
	}
});

</script>
