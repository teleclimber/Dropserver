<script setup lang="ts" >
import { ref } from 'vue';
import { storeToRefs } from 'pinia';

import { useAdminAllUsersStore } from '@/stores/admin/all_users';
import { useAdminUserInvitesStore } from '@/stores/admin/user_invitations';

import ViewWrap from '../../components/ViewWrap.vue';
import BigLoader from '@/components/ui/BigLoader.vue';
import UserListItem from '@/components/admin/UserListItem.vue';
import InviteListItem from '@/components/admin/InviteListItem.vue';
import InviteUser from '@/components/admin/InviteUser.vue';

const invites_store = useAdminUserInvitesStore();
invites_store.fetch();

const users_store = useAdminAllUsersStore();
const {users} = storeToRefs(users_store);
users_store.fetch();

const show_invite = ref(false);

</script>

<template>
	<ViewWrap>
		<div class="flex justify-between items-baseline mt-6 mb-2 mx-4 md:mx-0 ">
			<h2 class="text-xl font-bold">Invitations: ({{invites_store.invites.length}})</h2>
			<button @click="show_invite = true" v-if="!show_invite" class="btn btn-blue">Invite</button>
		</div>
		<InviteUser v-if="show_invite" @close="show_invite = false"></InviteUser>
		<InviteListItem	v-for="inv in invites_store.invites" :key="inv.email" :invite="inv"></InviteListItem>
		<div v-if="invites_store.invites.length === 0" class="bg-white px-4 py-1 text-gray-500 italic">No invitations.</div>

		<h2 class="text-xl font-bold mt-6 mb-2 ml-4 md:ml-0">Users: ({{users.size}})</h2>
		<UserListItem v-for="[_, user] in users" :key="user.value.user_id" :user="user.value"></UserListItem>
		<BigLoader v-if="!users_store.is_loaded"></BigLoader>
	</ViewWrap>
</template>

