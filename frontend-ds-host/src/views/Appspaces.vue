<template>
	<ViewWrap>
		<router-link to="new-appspace" class="btn btn-blue">New Appspace</router-link>

		<template v-if="appspaces.loaded">
			<AppspaceListItem v-for="a in appspaces.asArray" :key="a.id" :appspace="a"></AppspaceListItem>
		</template>
		<BigLoader v-else></BigLoader>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';

import { Appspaces } from '../models/appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import AppspaceListItem from '../components/AppspaceListItem.vue';

export default defineComponent({
	name: 'Appspaces',
	components: {
		ViewWrap,
		BigLoader,
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
