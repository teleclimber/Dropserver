<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useDropIDsStore } from '@/stores/dropids';

import { useAppspacesStore } from '@/stores/appspaces';
import { useRemoteAppspacesStore } from '@/stores/remote_appspaces';
import type { Appspace, RemoteAppspace } from '@/stores/types';

import ViewWrap from '@/components/ViewWrap.vue';
import BigLoader from '@/components/ui/BigLoader.vue';
import MessageSad from '@/components/ui/MessageSad.vue';
import AppspaceCard from '@/components/AppspaceCard.vue';

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const remoteAppspacesStore = useRemoteAppspacesStore();
remoteAppspacesStore.loadData();

onMounted( () => {
	appspacesStore.loadData();
	remoteAppspacesStore.loadData();
});

const dropIDStore = useDropIDsStore();
dropIDStore.loadData();

interface CardData {
	local:boolean,
	sort_string: string,
	local_appspace?: Appspace,
	remote_appspace?: RemoteAppspace
}
const asCards = computed( () => {
	const ret :CardData[] = [];
	if( appspacesStore.is_loaded ) {
		appspacesStore.appspaces.forEach( (a, id) => {
			ret.push({
				local: true,
				sort_string: a.value.domain_name,
				local_appspace: a.value
			});
		});
	}
	if( remoteAppspacesStore.is_loaded ) {
		remoteAppspacesStore.appspaces.forEach( (a) => {
			ret.push({
				local: false,
				sort_string: a.value.domain_name,
				remote_appspace: a.value
			});
		});
	}

	ret.sort( (a,b) => a.sort_string.localeCompare(b.sort_string) );

	return ret;
});

</script>

<template>
	<ViewWrap>
		<BigLoader v-if="!appspacesStore.is_loaded || !remoteAppspacesStore.is_loaded || !dropIDStore.is_loaded"></BigLoader>
		<template v-else>
			<div class="flex m-4 md:m-0 md:mb-6">
				<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
				<router-link to="new-remote-appspace" class="btn btn-blue">Join Remote Appspace</router-link>
			</div>

			<AppspaceCard v-for="a in asCards" 
				:key="a.sort_string"
				:local_appspace="a.local_appspace"
				:remote_appspace="a.remote_appspace"></AppspaceCard>

			<MessageSad head="No Appspaces"
				v-if="appspacesStore.appspaces.size === 0 && remoteAppspacesStore.appspaces.size === 0" 
				class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
				There are no appspaces in this account. Create or join one!
			</MessageSad>
		</template>

	</ViewWrap>
</template>

