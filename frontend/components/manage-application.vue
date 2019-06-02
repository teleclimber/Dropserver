<style scoped>
	.header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}

	.version {
		padding: 0.8rem 10px;
		border-bottom: 1px solid #ddd;
		background-color: white;
		cursor: pointer;
	}
	.version:hover {
		background-color: #ffa;
	}
	.version .ver-name {
		font-weight: bold;
	}
	.zero-versions {
		text-align: center;
		font-style: italic;
		color: #666;
		margin: 0 0 2em 0;
		padding: 0.8rem 10px;
		background-color: #ddd;
	}
	
	input.del-check {
		height: 2rem;
		padding: 0 0.2rem;
		margin: 0;
		box-sizing:border-box;
	}

	.submit {
		display: flex;
		justify-content: space-between;
		margin-top: 2rem;
	}
</style>

<template>
	<DsModal>
		<h2>Manage Application</h2>

		<p>Name: {{application.name}}</p>

		<div class="error" v-if="manage_status.state === 'error'">
			{{manage_status.error_message}}
		</div>
		<template v-else-if="cur_ver">
			<h4>Version {{cur_ver.name}}</h4>

			<p>{{cur_ver.num_use}} app spaces</p>
			<!-- wonder fif we could list these? -->

			<!-- show stats. if no app-spaces using it, offer a delete button -->
			<!-- stats? resource usage, and logs and errors? -->

			<DsButton @click="deleteVersion(cur_ver.name)" :disabled="cur_ver.num_use !== 0">delete version</DsButton>
		</template>
		<template v-else-if="manage_status.state === 'upload'">
			<p>Upload new version:</p>
			<UploadSelect @input="uploadSelectInput"></UploadSelect>
		</template>
		<template v-else-if="manage_status.state === 'uploading'">
			<p>Uploading...</p>
		</template>

		
		<template v-else>
			<div class="header">
				<p>{{application.versions.length}} versions</p>
				<DsButton @click="showUpload">Upload New Version</DsButton>
			</div>
			<div class="versions-container">
				<div 
						class="version"
						v-for="version in application.versions"
						:key="version.name"
						@click="cur_ver = version">
					<span class="ver-name">{{version.name}}</span>
					<span class="num-use">{{version.num_use}} app-spaces</span>
					<!-- could show latest version(?), number of app-spaces -->
				</div>
				<div v-if="application.versions.length == 0 " class="zero-versions">
					There are zero versions of this application :/
				</div>
			</div>

			<div class="delete">
				<p>Enter Name to delete:
				<input type="text" ref="del_check" class="del-check" @input="delCheckInput">
				<DsButton @click="doDeleteApplication" :disabled="!allow_delete">Delete</DsButton></p>
			</div>

		</template>

		<!-- manage application:
			- "Xkb across N version, n unused."
			- list versions, with # associated app-spaces, size it takes
			- delete version
			- upload new version
		-->

		<div class="submit">
			<DsButton @click="doClose" type="close">close</DsButton>
			<DsButton @click="doUpload" v-if="manage_status.state === 'upload'" :disabled="!upload_data">upload</DsButton>
		</div>
	</DsModal>
</template>

<script>
import DsModal from './ds-modal.vue';
import DsButton from './ds-button.vue';
import UploadSelect from './upload-select.vue';

export default {
	name: 'ManageApplication',
	components: {
		DsModal,
		DsButton,
		UploadSelect
	},
	data: function() {
		return {
			cur_ver: null,
			allow_delete: false,
			upload_data: null
		};
	},
	computed: {
		application: function() { 
			return this.$root.applications_vm.applications.find( a => a.name === this.manage_status.app_name )
		},
		manage_status: function() { return this.$root.applications_vm.manage_status; }
	},
	methods: {
		doClose: function() {
			if( this.cur_ver ) this.cur_ver = null;
			else this.$root.applications_vm.closeManageApplication();
		},
		uploadSelectInput: function( upload_data ) {
			this.upload_data = upload_data;
		},
		showUpload: function() {
			this.$root.applications_vm.showVersionUpload();
		},
		doUpload: function() {
			this.$root.applications_vm.uploadNewVersion( this.application.name, this.upload_data );
		},
		deleteVersion: function ( ver ) {
			this.$root.applications_vm.deleteVersion( this.application.name, ver )
			.then( () => {
				this.cur_ver = null;
			});
		},
		delCheckInput: function() {
			this.allow_delete = this.$refs.del_check.value.toLowerCase() === this.application.name.toLowerCase();
			return this.allow_delete;
		},
		doDeleteApplication: function() {
			if( this.delCheckInput() ) this.$root.applications_vm.deleteApplication( this.application.name );
		}
	}
}
</script>