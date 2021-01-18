<template>
  <div>
	
	<AppspaceListItem v-for="a in appspaces.asArray" :key="a.id" :appspace="a">
		
	</AppspaceListItem>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';

import axios from 'axios';
import type {AxiosResponse} from 'axios';

import { Resource } from '../utils/jsonapi_utils';

import { ReactiveAppspaces } from '../models/appspaces';

import AppspaceListItem from '../components/AppspaceListItem.vue';
import AppspaceApp from './AppspaceApp.vue';

export default defineComponent({
	name: 'Appspaces',
	components: {
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
