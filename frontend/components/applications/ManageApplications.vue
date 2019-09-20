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
			<p>Click an application to manage ({{applications_dm.applications.length}}):</p>
			<DsButton @click="applications_vm.createNew()">Add Application</DsButton>
		</div>
		<div class="apps-container">
			<div 
					class="application"
					v-for="application in applications_dm.applications"
					:key="application.app_id"
					@click="applications_vm.showManageApplication(application.app_id)">
				<span class="app-name">{{application.app_name}}</span>
				<span class="num-use">versions: {{ application.versions.length }}</span>
				<!-- could show latest version(?), number of app-spaces -->
			</div>
		</div>

		<div class="submit">
			<DsButton @click="applications_vm.listCloseClicked()" type="close">close</DsButton>
		</div>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import ApplicationsDM from '../../dms/applications-dm';

import ApplicationsVM from '../../vms/user-page/applications-vm';

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
	@Inject(ApplicationsDM.injectKey) readonly applications_dm!: ApplicationsDM;
	@Inject(ApplicationsVM.injectKey) readonly applications_vm!: ApplicationsVM;
};
</script>