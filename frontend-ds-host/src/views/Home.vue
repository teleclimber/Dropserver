<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useDropIDsStore } from '@/stores/dropids';

import { useAppspacesStore } from '@/stores/appspaces';
import type { Appspace } from '@/stores/types';

import ViewWrap from '@/components/ViewWrap.vue';
import BigLoader from '@/components/ui/BigLoader.vue';
import MessageSad from '@/components/ui/MessageSad.vue';
import AppspaceCard from '@/components/AppspaceCard.vue';

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

onMounted( () => {
	appspacesStore.loadData();
});

const dropIDStore = useDropIDsStore();
dropIDStore.loadData();

interface CardData {
	sort_string: string,
	local_appspace: Appspace,
}
const asCards = computed( () => {
	const ret :CardData[] = [];
	if( appspacesStore.is_loaded ) {
		appspacesStore.appspaces.forEach( (a, id) => {
			if( a.value.auth_user_id_conflicts === undefined ) return;	// implies no access
			ret.push({
				sort_string: a.value.domain_name,
				local_appspace: a.value
			});
		});
	}
	
	ret.sort( (a,b) => a.sort_string.localeCompare(b.sort_string) );

	return ret;
});

</script>

<template>
	<ViewWrap>
		<BigLoader v-if="!appspacesStore.is_loaded || !dropIDStore.is_loaded"></BigLoader>
		<template v-else>
			<div class="flex m-4 md:m-0 md:mb-6">
				<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
			</div>

			<AppspaceCard v-for="a in asCards" 
				:key="a.sort_string"
				:local_appspace="a.local_appspace"></AppspaceCard>

			<MessageSad head="No Appspaces"
				v-if="appspacesStore.appspaces.size === 0" 
				class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
				There are no appspaces in this account. Create or join one!
			</MessageSad>
		</template>

	</ViewWrap>
</template>

