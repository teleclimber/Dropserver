<script setup lang="ts">
import {reactive, onMounted, computed} from 'vue';
import { useDropIDsStore } from '@/stores/dropids';
import { Appspace, Appspaces } from '@/models/appspaces';
import {RemoteAppspaces, RemoteAppspace} from '@/models/remote_appspaces';

import ViewWrap from '@/components/ViewWrap.vue';
import BigLoader from '@/components/ui/BigLoader.vue';
import MessageSad from '@/components/ui/MessageSad.vue';
import AppspaceCard from '@/components/AppspaceCard.vue';

const dropIDStore = useDropIDsStore();
dropIDStore.loadData();

const no_dropids = computed( () => {
	return dropIDStore.is_loaded && dropIDStore.dropids.size === 0;
});

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
		<BigLoader v-if="!appspaces.loaded || !remote_appspaces.loaded || !dropIDStore.is_loaded"></BigLoader>
		<template v-else>
			<div class="bg-blue-100 py-5 flex mx-4 sm:mx-0 my-6 sm:rounded-xl shadow"
				v-if="dropIDStore.is_loaded && dropIDStore.dropids.size === 0">
				<div class="w-12 sm:w-16 flex flex-shrink-0 justify-center">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-8 h-8 text-blue-500">
						<path stroke-linecap="round" stroke-linejoin="round" d="M19 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zM4 19.235v-.11a6.375 6.375 0 0112.75 0v.109A12.318 12.318 0 0110.374 21c-2.331 0-4.512-.645-6.374-1.766z" />
					</svg>
				</div>
				<div class="pr-4 sm:pr-6 flex-grow">
					<h3 class="text-blue-600 text-lg font-medium pb-2">Create a DropID</h3>
					A DropID is how you identify yourself to an Appspace. Before going any further you should create one.
					<div class="flex justify-end mt-2">
						<router-link to="/dropid-new" class="btn">Go to the New DropID Page</router-link>
					</div>
				</div>
			</div>
			
			<div class="flex m-4 md:m-0 md:mb-6">
				<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
				<router-link to="new-remote-appspace" class="btn btn-blue">Join Appspace</router-link>
			</div>

			<AppspaceCard v-for="a in asCards" 
				:key="a.sort_string"
				:local_appspace="a.local_appspace"
				:remote_appspace="a.remote_appspace"></AppspaceCard>

			<MessageSad head="No Appspaces"
				v-if="appspaces.asArray.length === 0 && remote_appspaces.asArray.length === 0" 
				class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
				There are no appspaces in this account. Create or join one!
			</MessageSad>
		</template>

	</ViewWrap>
</template>

