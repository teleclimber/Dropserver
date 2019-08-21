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

<script>

import DsModal from './ds-modal.vue';
import DsButton from './ds-button.vue';
import UploadSelect from './upload-select.vue';

export default {
	name: 'CreateApplication',
	components: {
		DsModal,
		DsButton,
		UploadSelect
	},
	data: function() {
		return {
			vm: this.$root.applications_vm,	// application_vm
			upload_selected: null
		};
	},
	computed: {
		state: function() { 
			return this.vm.create_status.state;
		},
		show_upload_btn: function() { 
			return !this.state || this.state === 'uploading' || this.state === 'processing';
		},
		disable_upload_btn: function() {
			return !this.upload_selected || this.state === 'uploading' || this.state === 'processing';
		}
	},
	methods: {
		doClose: function() {
			// close if that's allowable.
			this.$root.cancelCreateApplication();
		},
		uploadSelectInput: function( form_data ) {
			this.upload_selected = form_data;
		},
		doUpload: function() {
			this.vm.createUpload( this.upload_selected );
		},
		doStartOver: function() {
			this.vm.createNew();
		},
		appNameChange: function() {
			const app_name = this.$refs.app_name_input.value;
			this.vm.appNameChanged( app_name );
		},
		openCreateAppSpace: function() {
			this.vm.openCreateAppSpace( this.vm.create_status.app_meta.app_name );
		}
	}
}
</script>