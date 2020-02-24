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

		<template v-if="create_vm.state === EditState.start">
			<UploadSelect :select_vm="create_vm.select_files_vm"></UploadSelect>
			<p v-if="create_vm.app_files_error">{{create_vm.app_files_error}}</p>
			<VersionComparison v-if="create_vm.version_comparison" :cmp="create_vm.version_comparison">
			</VersionComparison>

			<div class="submit">
				<DsButton @click="create_vm.doClose()" type="cancel">Cancel</DsButton>
				<span>
					<DsButton @click="create_vm.doUpload()" :disabled="!create_vm.enable_upload">Upload</DsButton>
				</span>
			</div>
		</template>

		<div class="error" v-if="create_vm.state === EditState.error">
			[ was create_vm.error_message ]

			<div class="submit">
				<DsButton @click="create_vm.doClose()" type="cancel">Cancel</DsButton>
				<span>
					<DsButton @click="create_vm.doStartOver()" >Start Over</DsButton>
				</span>
			</div>
		</div>

		<template v-if="create_vm.state === EditState.finished">
			<p>Application created</p>
			<p>{{create_vm.application.app_name}} @ {{create_vm.version_meta.version}}</p>
			<p>Customize application name, etc... [button]</p>
			<p>Create a new appspace for this application:</p>

			<div class="submit">
				<DsButton @click="create_vm.doClose()" type="cancel">Close</DsButton>
				<span>
					<DsButton @click="create_vm.createAppspaceClicked()">Create Appspace</DsButton>
				</span>
			</div>
		</template>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import CreateApplicationVM, { EditState } from '../../vms/user-page/create-application-vm';

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
export default class CreateApplication extends Vue {
	EditState = EditState;	// have to attach EditState to "this" so it can be used in template.

	@Prop({required: true, type: CreateApplicationVM}) readonly create_vm!: CreateApplicationVM;
}
</script>