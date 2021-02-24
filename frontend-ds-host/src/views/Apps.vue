<template>
	<ViewWrap>
		<router-link to="new-app" class="btn btn-blue">Upload New App</router-link>
		<AppListItem v-for="app in apps.asArray" :key="app.app_id" :app="app"></AppListItem>
		<BigLoader v-if="!apps.loaded"></BigLoader>
		<MessageSad v-else="apps.asArray.length === 0" head="No Applications" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			There are no applications in this account. Please upload one!
		</MessageSad>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent , onMounted, reactive } from 'vue';

import {Apps} from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import AppListItem from '../components/AppListItem.vue';

export default defineComponent({
	name: 'Apps',
	components: {
		ViewWrap,
		BigLoader,
		MessageSad,
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