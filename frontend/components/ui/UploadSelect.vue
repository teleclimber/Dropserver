<style scoped>

</style>

<template>
	<input type="file" name="app_dir" ref="app_dir" webkitdirectory @input="select_vm.setFileList(app_dir.files)" />
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import SelectAppFilesVM from '../../vms/ui/select-app-files-vm';

interface WebkitFile extends File {
	webkitRelativePath: string;
}

@Observer
@Component
export default class  UploadSelect extends Vue {
	@Prop({required: true, type:SelectAppFilesVM }) readonly select_vm!: SelectAppFilesVM;
	@Ref('app_dir') readonly app_dir!: HTMLInputElement;
};


// add support for:
// - upload single file (zip)
// - drop file or directory
// - pick by url? does that belong here?

</script>