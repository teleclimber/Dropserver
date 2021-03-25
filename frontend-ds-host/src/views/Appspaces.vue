<template>
	<ViewWrap>
		<div class="flex">
			<router-link to="new-appspace" class="btn btn-blue mr-2">Create Appspace</router-link>
			<router-link to="new-remote-appspace" class="btn btn-blue">Join Appspace</router-link>
		</div>

		<AppspaceListItem v-for="a in appspaces.asArray" :key="a.id" :appspace="a"></AppspaceListItem>
		<RemoteAppspaceListItem v-for="r in remote_appspaces.asArray" :key="r.domain_name" :remote_appspace="r"></RemoteAppspaceListItem>
		<BigLoader v-if="!appspaces.loaded || !remote_appspaces.loaded"></BigLoader>
		<MessageSad v-else-if="appspaces.asArray.length === 0 && remote_appspaces.asArray.length === 0" head="No Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			There are no appspaces in this account. Please create or join one!
		</MessageSad>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';

import { Appspaces } from '../models/appspaces';
import { RemoteAppspaces } from '../models/remote_appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import AppspaceListItem from '../components/AppspaceListItem.vue';
import RemoteAppspaceListItem from '../components/RemoteAppspaceListItem.vue';

export default defineComponent({
	name: 'Appspaces',
	components: {
		ViewWrap,
		BigLoader,
		MessageSad,
		AppspaceListItem,
		RemoteAppspaceListItem
	},
	setup() {
		const appspaces = reactive( new Appspaces );
		const remote_appspaces = reactive( new RemoteAppspaces );

		onMounted( async () => {
			appspaces.fetchForOwner();
			remote_appspaces.fetchForOwner();
		});
		
		return {
			appspaces,
			remote_appspaces
		};
	}
});

</script>
