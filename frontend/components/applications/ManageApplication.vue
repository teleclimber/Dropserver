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

		<p>Name: {{manage_vm.application.app_name}}</p>

		<div class="error" v-if="manage_vm.state === EditState.error">
			[ was manage_vm.error_message ]
		</div>
		<template v-else-if="manage_vm.show_version">
			<h4>Version {{manage_vm.show_version.version}}</h4>

			<p>[num??] app spaces</p>
			<p> List the appspaces</p>
			<!-- wonder fif we could list these? -->

			<!-- show stats. if no app-spaces using it, offer a delete button -->
			<!-- stats? resource usage, and logs and errors? -->

			<DsButton @click="manage_vm.deleteVersion(manage_vm.show_version.version)"
				>delete version</DsButton>
				<!-- :disabled="manage_vm.show_version.num_use !== 0" -->
		</template>
		<template v-else-if="manage_vm.state === EditState.upload">
			<p>Upload new version:</p>
			<UploadSelect :select_vm="manage_vm.select_files_vm"></UploadSelect>
			<p v-if="manage_vm.app_files_error">{{manage_vm.app_files_error}}</p>
			<VersionComparison v-if="manage_vm.version_comparison" :cmp="manage_vm.version_comparison">
			</VersionComparison>
		</template>
		<template v-else-if="manage_vm.state === EditState.uploading">
			<p>Uploading...</p>
		</template>

		<template v-else>
			<div class="header">
				<p>{{manage_vm.application.versions.length}} versions</p>
				<DsButton @click="manage_vm.showVersionUpload()">Upload New Version</DsButton>
			</div>
			<div class="versions-container">
				<div 
						class="version"
						v-for="version in manage_vm.application.versions"
						:key="version.version"
						@click="manage_vm.showVersion(version)">
					<span class="ver-name">{{version.version}}</span>
					<span class="num-use">?? app-spaces</span>
					<!-- could show latest version(?), number of app-spaces -->
				</div>
				<div v-if="manage_vm.application.versions.length == 0 " class="zero-versions">
					There are zero versions of this application :/
				</div>
			</div>

			<div class="delete">
				<p>Enter Name to delete:
				<input type="text" class="del-check" v-model="manage_vm.delete_check">
				<DsButton @click="manage_vm.deleteApplication()" :disabled="!manage_vm.allow_delete">Delete</DsButton></p>
			</div>

		</template>

		<!-- manage application:
			- "Xkb across N version, n unused."
			- list versions, with # associated app-spaces, size it takes
			- delete version
			- upload new version
		-->

		<div class="submit">
			<DsButton @click="manage_vm.closeClicked()" type="close">close</DsButton>
			<DsButton @click="manage_vm.uploadNewVersion()" v-if="manage_vm.state === EditState.upload" :disabled="!manage_vm.enable_upload">upload</DsButton>
		</div>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import ApplicationsDM from '../../dms/applications-dm';

import ApplicationsVM from '../../vms/user-page/applications-vm';
import { EditState, ManageApplicationVM } from '../../vms/user-page/applications-vm';

import VersionComparison from './VersionComparison.vue';
import DsModal from '../ui/DsModal.vue';
import DsButton from '../ui/DsButton.vue';
import UploadSelect from '../ui/UploadSelect.vue';

@Observer
@Component({
	components: {
		VersionComparison,
		DsModal,
		DsButton,
		UploadSelect
	}
})
export default class ManageApplication extends Vue {
	@Inject(ApplicationsVM.injectKey) readonly applications_vm!: ApplicationsVM;
	EditState = EditState;	// have to attach EditState to "this" so it can be used in template.

	@Prop({required: true, type: ManageApplicationVM}) readonly manage_vm!: ManageApplicationVM;
}
</script>