<style scoped>

</style>

<template>
	<div>
		<h1>Admin</h1>

		<p>
			{{ vm.registration }}
			<button @click="vm.showSettings">Settings</button>
			<button @click="vm.showInvitations">Invitations ({{vm.num_invitations}})</button>
		</p>

		<h2>Users: ({{vm.users.length}})</h2>

		<user v-for="(user,i) in vm.users" :key="i" :user="user"></user>

		<!-- button @click="ui.addUser">Add</button -->

		<DsModal v-if="vm.cur_modal">
			<component 
				:is="vm.cur_modal.is"
				:vm="vm.cur_modal"
			></component>
		</DsModal>
	</div>
</template>

<script>
import { observer } from "mobx-vue";

import User from "./components/User.vue";
import DsModal from '../../components/ds-modal.vue';
import AdminSettings from './components/AdminSettings.vue';
import AdminInvitations from './components/AdminInvitations.vue';

export default observer({
	name: 'Admin',
	components: {
		User,
		DsModal,
		AdminSettings,
		AdminInvitations,
	},
	computed: {
		vm: function() { return this.$root.vm; }
	}
});
</script>