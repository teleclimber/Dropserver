<style scoped>
	.submit {
		display: flex;
		justify-content: space-between;
		margin-top:2em;
	}
</style>

<template>
	<DsModal>
		<h2>New Application</h2>
		<!-- upload or pick from catalog or add as link -->

		<template v-if="!state">
			<UploadSelect @input="uploadSelectInput"></UploadSelect>
		</template>

		<div class="error" v-if="state === 'error'">
			{{vm.create_status.error_message}}
		</div>

		<template v-if="state === 'finished'">
			<p>Application created</p>
			<p>{{vm.create_status.app_meta.app_name}} @ {{vm.create_status.version_meta.version}}</p>
			<p>Customize application name, etc... [button]</p>
			<p>
				Create a new appspace for this application:
				<DsButton @click="openCreateAppSpace">Create Appspace</DsButton>
			</p>
		</template>

		<div class="submit">
			<DsButton @click="doClose" type="cancel">Cancel</DsButton>
			<span>
				<span class="state" v-if="state === 'uploading'">Uploading</span>
				<DsButton v-if="show_upload_btn" @click="doUpload" :disabled="disable_upload_btn">Upload</DsButton>
				<DsButton v-if="state == 'error'" @click="doStartOver" >Start Over</DsButton>
			</span>
		</div>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";

import DsModal from '../ui/DsModal.vue';
import DsButton from '../ui/DsButton.vue';
import UploadSelect from '../ui/UploadSelect.vue';

@Component({
	components: {
		DsModal,
		DsButton,
		UploadSelect
	}
})
export default class CreateApplication extends Vue {
	@Inject() readonly user_vm!: any;
	@Inject() readonly applications_vm!: any;

	upload_selected: any = null;

	@Ref('app_name_input') readonly app_name_input!: HTMLInputElement;

	get	state() { 
		return this.applications_vm.create_status.state;
	}
	get	show_upload_btn() { 
		return !this.state || this.state === 'uploading' || this.state === 'processing';
	}
	get	disable_upload_btn() {
		return !this.upload_selected || this.state === 'uploading' || this.state === 'processing';
	}
	
	doClose() {
		// close if that's allowable.
		this.user_vm.cancelCreateApplication();
	}
	uploadSelectInput( form_data: any ) {
		this.upload_selected = form_data;
	}
	doUpload() {
		this.applications_vm.createUpload( this.upload_selected );
	}
	doStartOver() {
		this.applications_vm.createNew();
	}
	appNameChange() {
		const app_name = this.app_name_input.value;
		this.applications_vm.appNameChanged( app_name );
	}
	openCreateAppSpace() {
		this.applications_vm.openCreateAppSpace( this.applications_vm.create_status.app_meta.app_name );
	}

}
</script>