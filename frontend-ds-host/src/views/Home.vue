<script setup lang="ts">
import {reactive, onMounted, computed} from 'vue';
import { Appspace, Appspaces } from '@/models/appspaces';
import {RemoteAppspaces, RemoteAppspace} from '@/models/remote_appspaces';

import ViewWrap from '@/components/ViewWrap.vue';
import BigLoader from '@/components/ui/BigLoader.vue';
import MessageSad from '@/components/ui/MessageSad.vue';
import AppspaceCard from '@/components/AppspaceCard.vue';

const appspaces = reactive( new Appspaces );
const remote_appspaces = reactive( new RemoteAppspaces );

onMounted( async () => {
	appspaces.fetchForOwner();
	remote_appspaces.fetchForOwner();
});

interface CardData {
	local:boolean,
	sort_string: string,
	local_appspace?: Appspace,
	remote_appspace?: RemoteAppspace
}
const asCards = computed( () => {
	const ret :CardData[] = [];
	if( appspaces.loaded ) {
		appspaces.as.forEach( (a, id) => {
			ret.push({
				local: true,
				sort_string: a.domain_name,
				local_appspace: a
			});
		});
	}
	if( remote_appspaces.loaded ) {
		remote_appspaces.remotes.forEach( (a) => {
			ret.push({
				local: false,
				sort_string: a.domain_name,
				remote_appspace: a
			});
		});
	}

	ret.sort( (a,b) => a.sort_string.localeCompare(b.sort_string) );

	return ret;
});

</script>

<template>
	<ViewWrap>
		<div class="flex m-4 md:m-0 md:mb-6">
			<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
			<router-link to="new-remote-appspace" class="btn btn-blue">Join Appspace</router-link>
		</div>

		<AppspaceCard v-for="a in asCards" 
			:key="a.sort_string"
			:local_appspace="a.local_appspace"
			:remote_appspace="a.remote_appspace"></AppspaceCard>

		<BigLoader v-if="!appspaces.loaded || !remote_appspaces.loaded"></BigLoader>
		<MessageSad v-else-if="appspaces.asArray.length === 0 && remote_appspaces.asArray.length === 0" head="No Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			There are no appspaces in this account. Please create or join one!
		</MessageSad>

	</ViewWrap>
</template>

