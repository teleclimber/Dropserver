<script setup lang="ts">
import { computed } from 'vue';

import AppspaceStatus from './AppspaceStatus.vue';
import Sandbox from './Sandbox.vue';

import baseData from '../models/base-data';
import appData from '../models/app-data';
import userData from '../models/user-data';

const active_user = computed( () => {
	return userData.getActiveUser();
});

</script>
<template>
	<header class="pt-4 px-4 flex justify-between">
		<div class="">
			<div class="flex items-baseline">
				<h1 class="text-2xl">{{appData.name}} <span class="bg-gray-300 px-2 rounded-md">{{appData.version}}</span></h1>
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