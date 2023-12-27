<script setup lang="ts">
import { onMounted } from 'vue';
import { useAppspacesStore } from '@/stores/appspaces';
import { useRemoteAppspacesStore } from '@/stores/remote_appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import AppspaceListItem from '../components/AppspaceListItem.vue';
import RemoteAppspaceListItem from '../components/RemoteAppspaceListItem.vue';
import { useAppsStore } from '@/stores/apps';

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const remoteAppspacesStore = useRemoteAppspacesStore();
remoteAppspacesStore.loadData();

const appsStore = useAppsStore();
appsStore.loadData();

onMounted( () => {
	appspacesStore.loadData();
	remoteAppspacesStore.loadData();
});

</script>

<template>
	<ViewWrap>
		<div class="flex m-4 md:m-0 md:mb-6">
			<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
			<router-link to="new-remote-appspace" class="btn btn-blue">Join Appspace</router-link>
		</div>

		<h2 class="text-xl font-bold mt-6 mb-2 ml-4 md:ml-0">Your Appspaces:</h2>
		<AppspaceListItem v-for="[_, a] in appspacesStore.appspaces" :key="a.value.appspace_id" :appspace="a.value"></AppspaceListItem>
		<BigLoader v-if="!appspacesStore.is_loaded"></BigLoader>
		<MessageSad v-else-if="appspacesStore.appspaces.size === 0" head="No Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			You have not created any appspaces.
			<router-link to="new-appspace" class="text-blue-700 underline">Create one</router-link>!
		</MessageSad>

		<h2 class="text-xl font-bold mt-6 mb-2 ml-4 md:ml-0">Remote Appspaces:</h2>
		<RemoteAppspaceListItem v-for="[_, r] in remoteAppspacesStore.appspaces" :key="r.value.domain_name" :remote_appspace="r.value"></RemoteAppspaceListItem>
		<BigLoader v-if="!remoteAppspacesStore.is_loaded"></BigLoader>
		<MessageSad v-else-if="remoteAppspacesStore.appspaces.size === 0" head="No Remote Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			You have not joined any remote appspaces. 
			<router-link to="new-remote-appspace" class="text-blue-700 underline">Join one</router-link>!
		</MessageSad>
	</ViewWrap>
</template>

