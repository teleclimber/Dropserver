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
		<!-- new app or manage existing -->
		<div class="header">
			<p>Click an application to manage:</p>
			<DsButton @click="showCreateApplication">Add Application</DsButton>
		</div>
		<div class="apps-container">
			<div 
					class="application"
					v-for="application in applications"
					:key="application.name"
					@click="showManageApplication(application.app_id)">
				<span class="app-name">{{application.app_name}}</span>
				<span class="num-use">nv: {{ application.versions.length }}</span>
				<!-- could show latest version(?), number of app-spaces -->
			</div>
		</div>

		<div class="submit">
			<DsButton @click="doClose" type="close">close</DsButton>
		</div>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";

import DsModal from './ds-modal.vue';
import DsButton from './ds-button.vue';

@Component({
	components: {
		DsModal,
		DsButton
	}
})
export default class ManageApplications extends Vue {
	@Inject() readonly user_vm!: any;
	@Inject() readonly applications_vm!: any;

	get applications() {
		return this.applications_vm.applications;
	}

	doClose() {
		this.user_vm.closeManageApplications();
	}
	showCreateApplication() {
		this.user_vm.showCreateApplication();
	}
	showManageApplication( app_id: any ) {
		console.log( 'manage'+app_id );
		this.applications_vm.showManageApplication( app_id );
	}
};
</script>