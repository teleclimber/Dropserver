<script lang="ts" setup>
import { useAppsStore } from '@/stores/apps';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import AppListItem from '../components/AppListItem.vue';

const appsStore = useAppsStore();
appsStore.loadData();
		
</script>

<template>
	<ViewWrap>
		<div class="flex m-4 md:m-0 md:mb-6">
			<router-link to="new-app" class="btn btn-blue">Get New App</router-link>
		</div>
		<AppListItem v-for="[app_id, app] in appsStore.apps" :key="app_id" :app="app.value"></AppListItem>
		<BigLoader v-if="!appsStore.is_loaded"></BigLoader>
		<MessageSad v-else-if="appsStore.apps.size === 0" head="No Applications" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			There are no applications in this account. Please upload one!
		</MessageSad>
	</ViewWrap>
</template>

