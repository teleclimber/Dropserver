<style scoped>
.user-container {
	padding: 0 10px 100px 10px;
}
.top-bar {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
.app-space-header {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
</style>

<template>
	<div class="user-container">
		<div class="top-bar">
			<h1>Hello.</h1>
			<div class="user-actions">
				<a v-if="user_vm.user.is_admin" :href="admin_url">[admin]</a>
				<a href="#" @click.prevent="showChangePassword" class="user-email">{{user_vm.user.email}}</a>
				<a :href="logout_url">logout</a>
			</div>
		</div>

		<div class="app-space-header">
			<h2>App Spaces: ({{app_spaces_vm.app_spaces.length}})</h2>
			<span>
				<DsButton @click="showManageApplications">Manage Applications</DsButton>
				<DsButton @click="showCreateAppSpace">New App Space</DsButton>
			</span>
		</div>
		<AppspaceListItem v-for="app_space in app_spaces_vm.app_spaces" :app_space="app_space" :key="app_space.id"></AppspaceListItem>

		<CreateAppspace 
			v-if="user_vm.ui.show_create_appspace">

		</CreateAppspace>

		<ManageAppspace
			v-if="user_vm.ui.show_manage_appspace">
		</ManageAppspace>

		<ManageApplications
			v-if="user_vm.ui.show_manage_applications">
		</ManageApplications>

		<CreateApplication
			v-if="user_vm.ui.show_create_application">
		</CreateApplication>

		<ManageApplication
			v-if="applications_vm.manage_status.app_id">
		</ManageApplication>

		<ChangePassword
			v-if="user_vm.ui.show_change_pw">
		</ChangePassword>
	</div>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";

import AppspaceListItem from './AppspaceListItem.vue';

import ManageApplications from '../applications/ManageApplications.vue';
import CreateApplication from '../applications/CreateApplication.vue';
import ManageApplication from '../applications/ManageApplication.vue';

import CreateAppspace from '../appspaces/CreateAppspace.vue';
import ManageAppspace from '../appspaces/ManageAppspace.vue';

import ChangePassword from './ChangePassword.vue';
import DsButton from '../ui/DsButton.vue';

declare global {
    interface Window { ds_user_routes_base_url: string; }
}

@Component({
	components: {
		AppspaceListItem,
		ManageAppspace,
		CreateAppspace,
		ManageApplications,
		ManageApplication,
		CreateApplication,
		ChangePassword,
		DsButton
	}
})
export default class UserPage extends Vue {
	@Inject() readonly user_vm!: any;
	@Inject() readonly applications_vm!: any;
	@Inject() readonly app_spaces_vm!: any;

	// get	ui() { return this.$root.ui; },
	// get user() { return this.$root.user; },

	get logout_url() { return window.ds_user_routes_base_url + "/logout" }
	get admin_url() { return window.ds_user_routes_base_url + "/admin" }

	showCreateAppSpace() {
		this.user_vm.showCreateAppSpace();
	}
	showManageApplications() {
		console.log("show managed applications");
		this.user_vm.showManageApplications();
	}
	showChangePassword() {
		this.user_vm.showChangePassword();
	}

}
</script>