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

<script lang="ts">
import { Vue, Component, Prop, Inject } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import AdminVM from "./vms/admin-root-vm";

import User from "./components/User.vue";
import DsModal from '../../components/ds-modal.vue';
import AdminSettings from './components/AdminSettings.vue';
import AdminInvitations from './components/AdminInvitations.vue';

@Observer
@Component({
	components: {
		User,
		DsModal,
		AdminSettings,
		AdminInvitations,
	}
})
export default class Admin extends Vue {
	@Inject() readonly vm!: AdminVM;
};
</script>