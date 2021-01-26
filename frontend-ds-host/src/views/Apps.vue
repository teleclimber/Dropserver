<template>
	<ViewWrap>
		<router-link to="new-app">Upload New App</router-link>
		<AppListItem v-for="app in apps.asArray" :key="app.app_id" :app="app"></AppListItem>
		
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent , onMounted } from 'vue';

import {ReactiveApps} from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import AppListItem from '../components/AppListItem.vue';

export default defineComponent({
	name: 'Apps',
	components: {
		ViewWrap,
		AppListItem
	},
	setup() {
		const apps = ReactiveApps();
		onMounted( async () => {
			await apps.fetchForOwner();
		});
		
		return {apps}
	}
});
</script>