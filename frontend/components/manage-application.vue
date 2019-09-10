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

		<p>Name: {{application.app_name}}</p>

		<div class="error" v-if="manage_status.state === 'error'">
			{{manage_status.error_message}}
		</div>
		<template v-else-if="cur_ver">
			<h4>Version {{cur_ver.version}}</h4>

			<p>[num??] app spaces</p>
			<p> List the appspaces</p>
			<!-- wonder fif we could list these? -->

			<!-- show stats. if no app-spaces using it, offer a delete button -->
			<!-- stats? resource usage, and logs and errors? -->

			<DsButton @click="deleteVersion(cur_ver.version)" :disabled="cur_ver.num_use !== 0">delete version</DsButton>
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
						:key="version.version"
						@click="cur_ver = version">
					<span class="ver-name">{{version.version}}</span>
					<span class="num-use">?? app-spaces</span>
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

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";

import DsModal from './ds-modal.vue';
import DsButton from './ds-button.vue';
import UploadSelect from './upload-select.vue';

@Component({
	components: {
		DsModal,
		DsButton,
		UploadSelect
	}
})
export default class ManageApplication extends Vue {
	@Inject() readonly user_vm!: any;
	@Inject() readonly applications_vm!: any;

	cur_ver: any = null;
	allow_delete: boolean = false;
	upload_data: any = null;

	@Ref('del_check') del_check!: HTMLInputElement;

	get application() { 
		return this.applications_vm.applications.find( (a: any) => a.app_id === this.manage_status.app_id )
	}
	get manage_status() { 
		return this.applications_vm.manage_status;
	}

	doClose() {
		if( this.cur_ver ) this.cur_ver = null;
		else this.applications_vm.closeManageApplication();
	}
	uploadSelectInput( upload_data: any ) {
		this.upload_data = upload_data;
	}
	showUpload() {
		this.applications_vm.showVersionUpload();
	}
	doUpload() {
		this.applications_vm.uploadNewVersion( this.application.app_id, this.upload_data );
	}
	deleteVersion( ver: string ) {
		this.applications_vm.deleteVersion( this.application.app_id, ver )
		.then( () => {
			this.cur_ver = null;
		});
	}
	delCheckInput() {
		this.allow_delete = this.del_check.value.toLowerCase() === this.application.app_name.toLowerCase();
		return this.allow_delete;
	}
	doDeleteApplication() {
		if( this.delCheckInput() ) this.applications_vm.deleteApplication( this.application.app_id );
	}
}
</script>