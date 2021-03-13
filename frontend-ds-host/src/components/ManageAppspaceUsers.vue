<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace Users</h3>
			<div class="flex items-stretch">
				<router-link class="btn" :to="{name:'appspace-new-user', params:{id:appspace.id}}">New user</router-link>
			</div>
		</div>
		<div class="divide-y divide-gray-200">
			<AppspaceUserListItem class="px-4 py-4 sm:px-6" v-for="user in appspace_users.au" :key="user.proxy_id" :appspace_id="appspace.id" :user="user"></AppspaceUserListItem>
		</div>
	</div>
	<!-- In future: select whether appspace is invite-only, or open registration -->
</template>


<script lang="ts">
import { defineComponent, ref, reactive, computed, onMounted, onUnmounted, PropType } from 'vue';

import type {App} from '../models/apps';
import type {Appspace} from '../models/appspaces';
import {AppspaceUsers} from '../models/appspace_users';

import AppspaceUserListItem from '../components/AppspaceUserListItem.vue';

export default defineComponent({
	name: 'ManageAppspaceUsers',
	components: {
		AppspaceUserListItem
	},
	props: {
		app: {
			type: Object as PropType<App>,
			required: true
		},
		appspace: {
			type: Object as PropType<Appspace>,
			required: true
		}
	},
	setup(props) {
		// Here we have to get the appspace id at the very least as a prop.
		const appspace_users = reactive(new AppspaceUsers);
		appspace_users.fetchForAppspace(props.appspace.id);

		return {
			appspace_users
		}


	}
});

</script>