<script setup lang="ts">
import {reactive, onMounted } from 'vue';

import { Appspaces } from '../models/appspaces';
import { RemoteAppspaces } from '../models/remote_appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import AppspaceListItem from '../components/AppspaceListItem.vue';
import RemoteAppspaceListItem from '../components/RemoteAppspaceListItem.vue';

const appspaces = reactive( new Appspaces );
const remote_appspaces = reactive( new RemoteAppspaces );

onMounted( async () => {
	appspaces.fetchForOwner();
	remote_appspaces.fetchForOwner();
});

</script>

<template>
	<ViewWrap>
		<div class="flex m-4 md:m-0 md:mb-6">
			<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
			<router-link to="new-remote-appspace" class="btn btn-blue">Join Appspace</router-link>
		</div>

		<h2 class="text-xl font-bold mt-6 mb-2 ml-4 md:ml-0">Your Appspaces:</h2>
		<AppspaceListItem v-for="a in appspaces.asArray" :key="a.id" :appspace="a"></AppspaceListItem>
		<BigLoader v-if="!appspaces.loaded"></BigLoader>
		<MessageSad v-else-if="appspaces.asArray.length === 0" head="No Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			You have not created any appspaces.
			<router-link to="new-appspace" class="text-blue-700 underline">Create one</router-link>!
		</MessageSad>

		<h2 class="text-xl font-bold mt-6 mb-2 ml-4 md:ml-0">Remote Appspaces:</h2>
		<RemoteAppspaceListItem v-for="r in remote_appspaces.asArray" :key="r.domain_name" :remote_appspace="r"></RemoteAppspaceListItem>
		<BigLoader v-if="!remote_appspaces.loaded"></BigLoader>
		<MessageSad v-else-if="remote_appspaces.asArray.length === 0" head="No Remote Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			You have not joined any remote appspaces. 
			<router-link to="new-remote-appspace" class="text-blue-700 underline">Join one</router-link>!
		</MessageSad>
	</ViewWrap>
</template>

