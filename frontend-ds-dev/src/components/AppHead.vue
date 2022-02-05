<template>
	<header class="pt-4 px-4 flex justify-between">
		<div class="">
			<div class="flex items-baseline">
				<h1 class="text-2xl">{{baseData.name}} <span class="bg-gray-300 px-2 rounded-md">{{baseData.version}}</span></h1>
				<p class="px-1">{{baseData.app_path}}</p>
			</div>
			<p>Appspace dir: 
				<span v-if="baseData.appspace_path">{{baseData.appspace_path}}</span>
				<span v-else class="text-gray-400 italic">(none provided)</span>
			</p>
		</div>
		<div class="flex items-stretch">
			<Sandbox></Sandbox>
			<div class="ml-4 flex flex-col items-stretch">
				<AppspaceStatus></AppspaceStatus>
				<div v-if="active_user" class="flex items-center bg-gray-100 mt-2 w-48">
					<img v-if="active_user.avatar" class="h-8 w-8" :src="'avatar/appspace/'+active_user.avatar">
					<span class="font-bold ml-2 whitespace-nowrap overflow-hidden flex-shrink">{{active_user.display_name}}</span>
				</div>
				<span v-else class="italic text-sm bg-gray-50 text-gray-600 mt-2 h-8 flex justify-center items-center">
					No user active.
				</span>
			</div>
		</div>		
	</header>
</template>

<script lang="ts">
import { defineComponent, computed } from 'vue';

import AppspaceStatus from './AppspaceStatus.vue';
import Sandbox from './Sandbox.vue';

import baseData from '../models/base-data';
import appspaceStatus from '../models/appspace-status';
import userData from '../models/user-data';

export default defineComponent({
	name: 'AppHead',
	components: {
		AppspaceStatus,
		Sandbox
	},
	setup(props, context) {

		const active_user = computed( () => {
			return userData.getActiveUser();
		});
		return {
			baseData, appspaceStatus,
			active_user
		}
	}
});
</script>