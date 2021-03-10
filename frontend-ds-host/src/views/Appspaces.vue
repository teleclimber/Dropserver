<template>
	<ViewWrap>
		<router-link to="new-appspace" class="btn btn-blue">New Appspace</router-link>

		<AppspaceListItem v-for="a in appspaces.asArray" :key="a.id" :appspace="a"></AppspaceListItem>
		<BigLoader v-if="!appspaces.loaded"></BigLoader>
		<MessageSad v-else-if="appspaces.asArray.length === 0" head="No Appspaces" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			There are no appspaces in this account. Please create one!
		</MessageSad>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';

import { Appspaces } from '../models/appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import AppspaceListItem from '../components/AppspaceListItem.vue';

export default defineComponent({
	name: 'Appspaces',
	components: {
		ViewWrap,
		BigLoader,
		MessageSad,
		AppspaceListItem
	},
	setup() {
		const appspaces = reactive( new Appspaces );
		onMounted( async () => {
			await appspaces.fetchForOwner();
		});
		
		return {appspaces}
	}
});

</script>
