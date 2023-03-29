<script lang="ts" setup>
import { computed } from 'vue';

import { useAppspaceUsersStore } from '@/stores/appspace_users';

import AppspaceUserListItem from '../components/AppspaceUserListItem.vue';

const props = defineProps<{
	appspace_id: number
}>();

const appspaceUsersStore = useAppspaceUsersStore();
appspaceUsersStore.loadData(props.appspace_id);
const appspace_users = computed( () => {
	if( appspaceUsersStore.isLoaded(props.appspace_id) ) return appspaceUsersStore.mustGetUsers(props.appspace_id).value;
})

</script>

<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace Users</h3>
			<div class="flex items-stretch">
				<router-link class="btn" :to="{name:'appspace-new-user', params:{appspace_id:appspace_id}}">New user</router-link>
			</div>
		</div>
		<div v-if="appspace_users" class="divide-y divide-gray-200">
			<AppspaceUserListItem class="px-4 py-4 sm:px-6" v-for="user in appspace_users" :key="user.value.proxy_id" :appspace_id="appspace_id" :user="user.value"></AppspaceUserListItem>
		</div>
	</div>
</template>
