<template>
	<ViewWrap>
		<router-link to="new-app" class="btn btn-blue">Upload New App</router-link>
		<AppListItem v-for="app in apps.asArray" :key="app.app_id" :app="app"></AppListItem>
		<BigLoader v-if="!apps.loaded"></BigLoader>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent , onMounted, reactive } from 'vue';

import {Apps} from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import AppListItem from '../components/AppListItem.vue';

export default defineComponent({
	name: 'Apps',
	components: {
		ViewWrap,
		BigLoader,
		AppListItem
	},
	setup() {
		const apps = reactive(new Apps);
		onMounted( async () => {
			await apps.fetchForOwner();
		});
		
		return {apps}
	}
});
</script>