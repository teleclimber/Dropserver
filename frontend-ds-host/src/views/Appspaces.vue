<script setup lang="ts">
import { onMounted, computed } from 'vue';
import { useAppspacesStore } from '@/stores/appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import OwnedAppspaceListItem from '../components/OwnedAppspaceListItem.vue';
import AppspaceListItem from '../components/AppspaceListItem.vue';
import { useAppsStore } from '@/stores/apps';
import { useAuthUserStore } from '@/stores/auth_user';
import type { Appspace } from '@/stores/types';

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const appsStore = useAppsStore();
appsStore.loadData();

onMounted( () => {
	appspacesStore.loadData();
});

const appspaces = computed( () => {
	const ret :{owned: Appspace[], other:Appspace[]} = {owned:[], other:[]};
	appspacesStore.appspaces.forEach( a => {
		if( a.value.owner_id === authUserStore.user_id ) ret.owned.push(a.value);
		else ret.other.push(a.value);
	});
	return ret;
});

</script>

<template>
	<ViewWrap>
		<div class="flex m-4 md:m-0 md:mb-6">
			<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
		</div>

		<h2 class="text-xl font-bold mt-6 mb-2 ml-4 md:ml-0">Your Appspaces:</h2>
		<OwnedAppspaceListItem v-for="a in appspaces.owned" :key="a.appspace_id" :appspace="a"></OwnedAppspaceListItem>
		<BigLoader v-if="!appspacesStore.is_loaded"></BigLoader>
		<MessageSad v-else-if="appspaces.owned.length === 0" head="No Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			You have not created any appspaces.
			<router-link to="new-appspace" class="text-blue-700 underline">Create one</router-link>!
		</MessageSad>

		<h2 class="text-xl font-bold mt-6 mb-2 ml-4 md:ml-0">Appspaces You Can Access:</h2>
		<AppspaceListItem v-for="a in appspaces.other" :key="a.appspace_id" :appspace="a"></AppspaceListItem>
		<BigLoader v-if="!appspacesStore.is_loaded"></BigLoader>
		<MessageSad v-else-if="appspaces.other.length === 0" head="No Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			You do not have access to other appspaces on this instance.
		</MessageSad>
	</ViewWrap>
</template>

