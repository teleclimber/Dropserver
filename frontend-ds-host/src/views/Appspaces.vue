<template>
  <ViewWrap>
	
	<AppspaceListItem v-for="a in appspaces.asArray" :key="a.id" :appspace="a">
		
	</AppspaceListItem>
  </ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';

import { ReactiveAppspaces } from '../models/appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import AppspaceListItem from '../components/AppspaceListItem.vue';

export default defineComponent({
	name: 'Appspaces',
	components: {
		ViewWrap,
		AppspaceListItem
	},
	setup() {
		const appspaces = ReactiveAppspaces();
		onMounted( async () => {
			await appspaces.fetchForOwner();
		});
		
		return {appspaces}
	}
});

</script>
