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
				<a v-if="user.is_admin" :href="admin_url">[admin]</a>
				<a href="#" @click.prevent="showChangePassword" class="user-email">{{user.email}}</a>
				<a :href="logout_url">logout</a>
			</div>
		</div>

		<div class="app-space-header">
			<h2>App Spaces: ({{app_spaces.length}})</h2>
			<span>
				<DsButton @click="showManageApplications">Manage Applications</DsButton>
				<DsButton @click="showCreateAppSpace">New App Space</DsButton>
			</span>
		</div>
		<app-space v-for="app_space in app_spaces" :app_space="app_space" :key="app_space.id"></app-space>

		<CreateAppSpace 
			v-if="ui.show_create_appspace">

		</CreateAppSpace>

		<ManageAppSpace
			v-if="ui.show_manage_appspace">
		</ManageAppSpace>

		<ManageApplications
			v-if="ui.show_manage_applications">
		</ManageApplications>

		<CreateApplication
			v-if="ui.show_create_application">
		</CreateApplication>

		<ManageApplication
			v-if="applications_vm.manage_status.app_id">
		</ManageApplication>

		<ChangePassword
			v-if="ui.show_change_pw">
		</ChangePassword>
	</div>
</template>

<script>

import AppSpace from './AppSpace.vue';
import ManageAppSpace from '../../components/manage-appspace.vue';
import ManageApplications from '../../components/manage-applications.vue';
import ManageApplication from '../../components/manage-application.vue';
import CreateAppSpace from '../../components/create-appspace.vue';
import CreateApplication from '../../components/create-application.vue';
import ChangePassword from './ChangePassword.vue';
import DsButton from '../../components/ds-button.vue';

export default {
	computed: {
		ui: function() { return this.$root.ui; },
		user: function() { return this.$root.user; },
		app_spaces: function() { return this.$root.app_spaces_vm.app_spaces; },
		applications_vm: function() { return this.$root.applications_vm; },
		logout_url: function() { return window.ds_user_routes_base_url + "/logout" },
		admin_url: function() { return window.ds_user_routes_base_url + "/admin" }
	},
	components: {
		'app-space': AppSpace,
		ManageAppSpace,
		CreateAppSpace,
		ManageApplications,
		ManageApplication,
		CreateApplication,
		ChangePassword,
		DsButton
	},
	methods: {
		showCreateAppSpace: function() {
			this.$root.showCreateAppSpace();
		},
		showManageApplications: function() {
			this.$root.showManageApplications();
		},
		showChangePassword: function() {
			this.$root.showChangePassword();
		}
	}
}


</script>