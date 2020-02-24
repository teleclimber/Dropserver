<style scoped>
	.header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}
	.application {
		padding: 0.8em 10px;
		border-bottom: 1px solid #ddd;
		background-color: white;
		cursor: pointer;
	}
	.application:hover {
		background-color: #ffa;
	}
	.application .app-name {
		font-weight: bold;
	}
	.submit {
		margin-top: 2rem;
	}
</style>

<template>
	<DsModal>
		<h2>Manage Applications</h2>
		<div class="header">
			<p>Click an application to manage ({{list_apps_vm.sorted_apps.length}}):</p>
			<DsButton @click="applications_ui.createNew()">Add Application</DsButton>
		</div>
		<div class="apps-container">
			<div 
					class="application"
					v-for="application in list_apps_vm.sorted_apps"
					:key="application.app_id"
					@click="applications_ui.showManageApplication(application.app_id)">
				<span class="app-name">{{application.app_name}}</span>
				<span class="num-use">
					{{ application.versions.length }} versions
					{{ list_apps_vm.app_uses[application.app_id].num_appspace }} appspaces
				</span>
				
				<!-- could show latest version(?), number of app-spaces -->
			</div>
		</div>

		<div class="submit">
			<DsButton @click="applications_ui.listCloseClicked()" type="close">close</DsButton>
		</div>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import ListApplicationsVM from '../../vms/user-page/list-applications-vm';
import ApplicationsUI from '../../vms/user-page/applications-ui';

import DsModal from '../ui/DsModal.vue';
import DsButton from '../ui/DsButton.vue';

@Observer
@Component({
	components: {
		DsModal,
		DsButton
	}
})
export default class ManageApplications extends Vue {	// This should really be called ListApplications
	@Inject(ListApplicationsVM.injectKey) readonly list_apps_vm!: ListApplicationsVM;
	@Inject(ApplicationsUI.injectKey) readonly applications_ui!: ApplicationsUI;
};
</script>