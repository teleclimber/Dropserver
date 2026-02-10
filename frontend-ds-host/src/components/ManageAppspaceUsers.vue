<script lang="ts" setup>
import { computed, onMounted } from 'vue';

import { useAppspaceUsersStore } from '@/stores/appspace_users';
import { useAppspaceUserConflictsStore } from '@/stores/appspace_user_conflicts';

import AppspaceUserListItem from '../components/AppspaceUserListItem.vue';
import MessageWarn from './ui/MessageWarn.vue';

const props = defineProps<{
	appspace_id: number
}>();

const appspaceUsersStore = useAppspaceUsersStore();
appspaceUsersStore.loadData(props.appspace_id);

const userConflicts = useAppspaceUserConflictsStore();
const conflicts = computed( () => {
	return userConflicts.getForAppspace(props.appspace_id).value;
});

onMounted( () => {
	appspaceUsersStore.reloadData(props.appspace_id);
});
const appspace_users = computed( () => {
	if( appspaceUsersStore.isLoaded(props.appspace_id) ) return appspaceUsersStore.mustGetUsers(props.appspace_id);
});

const multi_user_conflicts = computed( () => {
	const ret :Set<number> = new Set;
	conflicts.value.forEach( (c, p) => {
		if( c.conflict && c.proxy_id_matches.size > 1 ) {
			const user_id = Array.from(c.user_id_matches.keys()).pop();
			if( user_id ) ret.add(user_id);
		}
	});
	return ret;
});

</script>

<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace Users</h3>
			<div class="flex items-stretch">
				<router-link class="btn" :to="{name:'appspace-new-user', params:{appspace_id:appspace_id}}">New user</router-link>
			</div>
		</div>
		<MessageWarn head="User Conflicts" v-if="multi_user_conflicts.size">
			Some users of this Dropserver instance are matching with multiple users of this Appspace.
		</MessageWarn>
		<div v-if="appspace_users" class="divide-y divide-gray-200">
			<AppspaceUserListItem class="px-4 py-4 sm:px-6" v-for="user in appspace_users" 
				:key="user.proxy_id" 
				:appspace_id="appspace_id" 
				:user="user"
				:conflicts="conflicts.get(user.proxy_id)"></AppspaceUserListItem>
		</div>
	</div>
</template>
