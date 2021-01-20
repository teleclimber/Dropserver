<template>
	<ViewWrap>
		<h3>Manage appspace</h3>

		<p>Subdomain: {{appspace.subdomain}}</p>
		<p>Created {{appspace.created_dt.toLocaleString()}}</p>
		<p>App: {{appspace.app_version.name}}, version: {{appspace.app_version.version}}</p>
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

import { Resource } from '../utils/jsonapi_utils';

import { ReactiveAppspace, Appspace } from '../models/appspaces';

import ViewWrap from '../components/ViewWrap.vue';

export default defineComponent({
	name: 'ManageAppspace',
	components: {
		ViewWrap
	},
	setup() {
		const route = useRoute();
		const appspace = ReactiveAppspace();
		onMounted( async () => {
			await appspace.fetch(Number(route.params.id));
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
			pause,
			unPause,
			pausing,
		};
	}
});

</script>
