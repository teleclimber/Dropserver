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
				<a v-if="current_user_dm.user && current_user_dm.user.is_admin" :href="admin_url">[admin]</a>
				<a href="#" v-if="current_user_dm.user" @click.prevent="user_page_vm.showChangePassword()" class="user-email">{{current_user_dm.user.email}}</a>
				<a :href="logout_url">logout</a>
			</div>
		</div>

		<div class="app-space-header">
			<h2>Appspaces: ({{appspaces_dm.appspaces.length}})</h2>
			<span>
				<DsButton @click="user_page_vm.showApplicationsList()">Manage Applications</DsButton>
				<DsButton @click="user_page_vm.showCreateAppspace()">New Appspace</DsButton>
			</span>
		</div>
		<AppspaceListItem 
			v-for="appspace in appspaces_dm.appspaces" 
			:appspace_vm="appspaces_vm.getAugmentedAppspace(appspace)" 
			:key="appspace.appspace_id"></AppspaceListItem>

		<CreateAppspace 
			v-if="appspaces_vm.create_appspace_vm"
			:create_vm="appspaces_vm.create_appspace_vm">
		</CreateAppspace>

		<ManageAppspace
			v-if="appspaces_vm.manage_appspace_vm"
			:manage_vm="appspaces_vm.manage_appspace_vm">
		</ManageAppspace>

		<ManageApplications
			v-if="applications_ui.show_list">
		</ManageApplications>

		<CreateApplication
			v-if="applications_ui.create_vm"
			:create_vm="applications_ui.create_vm">
		</CreateApplication>

		<ManageApplication
			v-if="applications_ui.manage_vm"
			:manage_vm="applications_ui.manage_vm">
		</ManageApplication>

		<ChangePassword
			v-if="user_page_vm.change_pw_vm"
			:change_pw_vm="user_page_vm.change_pw_vm">
		</ChangePassword>
	</div>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import CurrentUserDM from '../../dms/current-user-dm';
import AppspacesDM from '../../dms/appspaces-dm';

import UserPageUI from "../../vms/user-page/user-page-ui";
import AppspacesUI from '../../vms/user-page/appspaces-ui';
import ApplicationsUI from "../../vms/user-page/applications-ui";

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

@Observer
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
	@Inject(CurrentUserDM.injectKey) readonly current_user_dm!: CurrentUserDM;
	@Inject(AppspacesDM.injectKey) readonly appspaces_dm!: AppspacesDM;

	@Inject(UserPageUI.injectKey) readonly user_page_vm!: UserPageUI;
	@Inject(AppspacesUI.injectKey) readonly appspaces_vm!: AppspacesUI;
	@Inject(ApplicationsUI.injectKey) readonly applications_ui!: ApplicationsUI;
	
	get logout_url() { return window.ds_user_routes_base_url + "/logout" }
	get admin_url() { return window.ds_user_routes_base_url + "/admin" }
}
</script>