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

		<template v-if="state === 'enter-meta'">
			<label for="app-name">Application name:</label>
			<input 
				type="text"
				id="app-name"
				:value="vm.create_status.cur_name"
				@input="appNameChange"
				ref="app_name_input">
			<span>{{name_status}}</span>
			<p>version: {{vm.create_status.app_meta.version}}</p>
		</template>

		<template v-if="state === 'finished'">
			<p>
				Application created
				<DsButton @click="openCreateAppSpace">Create App Space</DsButton>
			</p>
		</template>

		<div class="submit">
			<DsButton @click="doClose" type="cancel">Cancel</DsButton>
			<span>
				<span class="state" v-if="state === 'uploading'">Uploading</span>
				<DsButton v-if="show_next_btn" @click="doNext" :disabled="disable_next_btn">Next</DsButton>
				<DsButton v-if="state == 'error'" @click="doStartOver" >Start Over</DsButton>
				<DsButton v-if="show_finish_btn" @click="doFinish" :disabled="disable_finish_btn">Finish</DsButton>
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
		show_next_btn: function() { 
			return !this.state || this.state === 'uploading' || this.state === 'processing';
		},
		disable_next_btn: function() {
			return this.state === 'uploading' || this.state === 'processing';
		},
		show_finish_btn: function() {
			return this.state === 'enter-meta' || this.state === 'finishing';
		},
		disable_finish_btn: function() {
			return this.state === 'finishing' || this.name_status !== 'available';
		},
		name_status: function() {
			const cur_name = this.vm.create_status.cur_name;
			const name_available = this.vm.create_status.name_available;
			if( !(cur_name in name_available) ) return 'checking';
			else if( name_available[cur_name] ) return 'available';
			else return 'unavailable';

			// Also gotta do invalid names (too short, too long, bad chars)
			// -> requires client side validation lib.
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
		doNext: function() {
			this.vm.createDoNext( this.upload_selected );
		},
		doStartOver: function() {
			this.vm.createNew();
		},
		appNameChange: function() {
			const app_name = this.$refs.app_name_input.value;
			this.vm.appNameChanged( app_name );
		},
		doFinish: function() {
			// collect meta: name, auto-fetch updates, etc...
			this.vm.createFinish({
				name: this.$refs.app_name_input.value
			});
		},
		openCreateAppSpace: function() {
			this.vm.openCreateAppSpace( this.vm.create_status.app_meta.app_name );
		}
	}
}
</script>