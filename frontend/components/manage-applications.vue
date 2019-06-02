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
					@click="showManageApplication(application.name)">
				<span class="app-name">{{application.name}}</span>
				<span class="num-use">{{ application.versions.map( ver => ver.num_use ).reduce( (sum,num) => sum + num, 0 ) }}</span>
				<!-- could show latest version(?), number of app-spaces -->
			</div>
		</div>

		<div class="submit">
			<DsButton @click="doClose" type="close">close</DsButton>
		</div>
	</DsModal>
</template>

<script>
import DsModal from './ds-modal.vue';
import DsButton from './ds-button.vue';

export default {
	name: 'ManageApplications',
	components: {
		DsModal,
		DsButton
	},
	computed: {
		applications: function() { return this.$root.applications_vm.applications; }
	},
	methods: {
		doClose: function() {
			this.$root.closeManageApplications();
		},
		showCreateApplication: function() {
			this.$root.showCreateApplication();
		},
		showManageApplication: function( app_name ) {
			console.log( 'manage'+app_name );
			this.$root.applications_vm.showManageApplication( app_name );
		}
	}
}
</script>