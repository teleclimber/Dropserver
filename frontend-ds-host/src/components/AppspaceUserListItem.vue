<template>
	<div>
		<div class="pb-1 flex items-baseline justify-between">
			<div class="flex flex-col sm:flex-row items-baseline">
				<span class="pr-2 font-bold text-xl">{{user.display_name}}</span>
				<div>
					<span v-for="p in user.permissions" :key="p">{{p}}</span>
					<span class="italic text-gray-400" v-if="user.permissions.length === 0">No permissions</span>
				</div>
			</div>
			<router-link class="btn" :to="{name:'appspace-user', params:{id: appspace_id, proxy_id:user.proxy_id}}">Edit</router-link>
		</div>
		<div class="text-gray-600">{{user.auth_id}}</div>
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
