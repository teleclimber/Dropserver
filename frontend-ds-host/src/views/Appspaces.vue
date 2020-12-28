<template>
  <div>
	
	<div v-for="a in appspaces.asArray" :key="a.appspace_id">
		<h2 class="text-xl">{{a.subdomain}}</h2>
		<AppspaceApp :app_id="a.app_id" :version="a.app_version" />
	
	</div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import appspaces from '../models/appspaces';

import AppspaceApp from './AppspaceApp.vue';

export default defineComponent({
	name: 'Appspaces',
	components: {
		AppspaceApp
	},
	setup() {
		appspaces.fetchAll();
		return {appspaces}
	}
});
</script>
