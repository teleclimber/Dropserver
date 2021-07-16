<template>
	<div class="flex">
		<img v-if="user.avatarURL" :src="user.avatarURL" class="w-12 h-12 flex-shrink-0 rounded-full bg-clip-border">
		<div v-else class="w-12 h-12 flex-shrink-0 rounded-full bg-clip-border bg-gray-200">&nbsp;</div>
		<div class="flex-grow flex-shrink pl-4 overflow-hidden">
			<div class="flex flex-col sm:flex-row items-baseline">
				<span class="pr-2 font-bold text-l">{{user.display_name}}</span>
				<div>
					<span v-for="p in user.permissions" :key="p">{{p}}</span>
					<span class="italic text-gray-400" v-if="user.permissions.length === 0">No permissions</span>
				</div>
			</div>
			<div class="text-gray-600">{{user.auth_id}}</div>
		</div>
		<router-link class="btn self-start flex-shrink-0" :to="{name:'appspace-user', params:{id: appspace_id, proxy_id:user.proxy_id}}">Edit</router-link>
		
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {AppspaceUser} from '../models/appspace_users';

export default defineComponent({
	name: 'AppspaceUserListItem',
	components: {
	},
	props: {
		appspace_id: {
			type: Number,
			required: true
		},
		user: {
			type: Object as PropType<AppspaceUser>,
			required: true
		}
	},
	setup(props) {

	}
});
</script>
