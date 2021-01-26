<template>
	<ViewWrap>
		<h3>{{app.name}}</h3>

		<p>Created {{app.created_dt.toLocaleString()}}</p>

		<router-link :to="{name: 'new-app-version', params:{id:app.app_id}}">Upload New Version</router-link>
		
		<ul>
			<li v-for="ver in app.versions" :key="ver.version">{{ver.version}} created {{ver.created_dt.toLocaleString()}}</li>
		</ul>
	</ViewWrap>
</template>


<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, reactive, onMounted } from 'vue';

import { App } from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';

export default defineComponent({
	name: 'ManageApp',
	components: {
		ViewWrap
	},
	setup() {
		const route = useRoute();
		const app = reactive(new App);
		onMounted( async () => {
			await app.fetch(Number(route.params.id));
		});

		return {
			app,
		};
	}
});

</script>