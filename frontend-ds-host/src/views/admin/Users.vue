<template>
	<ViewWrap>
		<h2>Invitations: ({{invitations.invitations.size}}) <button @click="showInvite">Invite</button></h2>
		<p v-if="show_invite">
			Email: <input type="text" ref="email_input" v-model="invite_email">
			<button @click="cancelInvite">Cancel</button>
			<button @click="saveInvite">Save</button>
		</p>
		<p v-for="inv in invitations.asArray" :key="inv.email">{{inv.email}}</p>

		<h2>Users: ({{admin_users.users.size}})</h2>
		<p v-for="user in admin_users.asArray" :key="user.user_id">{{user.user_id}} {{user.email}} {{user.is_admin ? "admin" : ""}}</p>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, reactive, ref, Ref, nextTick } from 'vue';

import {AdminUsers} from '../../models/user';
import {AdminInvitations} from '../../models/admin_invitations';

import ViewWrap from '../../components/ViewWrap.vue';

export default defineComponent({
	name: 'Users',
	components: {
		ViewWrap
	},
	setup() {
		const admin_users = reactive(new AdminUsers);
		admin_users.fetch();

		const invitations = reactive(new AdminInvitations);
		invitations.fetch();

		const show_invite = ref(false);
		const email_input :Ref<HTMLInputElement|undefined> = ref(undefined);
		const invite_email = ref("");
		function showInvite() {
			show_invite.value = true;
			nextTick( () => email_input.value?.focus() );
		}
		function cancelInvite() {
			show_invite.value = false;
		}
		async function saveInvite() {
			if( email_input.value === undefined ) return;
			const email = invite_email.value.trim();
			if( !email.includes("@") || email.length < 5 ) return;
			await invitations.createInvitation( invite_email.value );
			show_invite.value = false;
		}

		return {
			admin_users,
			invitations,
			showInvite,
			cancelInvite,
			saveInvite,
			show_invite,
			invite_email,
			email_input
		}
	}
});
</script>
