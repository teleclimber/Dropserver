<template>
	<ViewWrap>
		<template v-if="step === 'pick-app'">
			<h3>Pick Application and Version</h3>

			<AppListItem v-for="app in apps.asArray" :key="app.app_id" :app="app"></AppListItem>

		</template>

		<template v-if="step === 'settings'">
			<h3>New Appspace Settings</h3>

			<p>{{app_version.name}} {{app_version.version}}</p>


			<p>domain, subdomain, etc..</p>

			<button @click="create">Create</button>
		</template>
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, watchEffect } from 'vue';
import {useRoute} from 'vue-router';
import router from '../router';

import { ReactiveApps } from '../models/apps';
import { ReactiveAppVersion } from '../models/app_versions';
import {createAppspace} from '../models/appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import AppListItem from '../components/AppListItem.vue';

export default defineComponent({
	name: 'NewAppspace',
	components: {
		ViewWrap,
		AppListItem
	},
	setup() {
		const route = useRoute();
		const step = ref("wait");	// wait, pick-app, settings

		const app_id = ref(0);
		const version = ref('');

		const apps = ReactiveApps();
		const app_version = ReactiveAppVersion();

		watchEffect(() => {
			if( route.query.app_id !== undefined && typeof route.query.version === 'string' && route.query.version !== '' ) {
				app_id.value = Number(route.query.app_id);
				version.value = route.query.version;
				app_version.fetch(app_id.value, version.value);
				step.value = 'settings';
			}
			else {
				apps.fetchForOwner();
				step.value = 'pick-app';
			}
		});

		async function create() {
			const appspace_id = await createAppspace(app_id.value, version.value);
			router.push({name: 'manage-appspace', params:{id: appspace_id+''}});
		}

		return {
			step,
			apps,
			app_version,
			create,
		};
	},
});

</script>