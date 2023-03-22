<script lang="ts" setup>
import {Ref, ref} from 'vue';
import type { SelectedFile } from '@/stores/types';

interface WebkitFile extends File {
	webkitRelativePath: string;
}

const emit = defineEmits<{
  (e: 'changed', files:SelectedFile[]):void
}>()

const input_elem :Ref<HTMLInputElement|null> = ref(null);
function selected() {
	if( input_elem.value === null ) return;

	const files = input_elem.value.files as FileList;
	const prefix = getPrefix(files);
	const chop_length = prefix ? prefix.length +1 : 0;

	const selected_files: SelectedFile[] = [];

	for( let i=0; i<files.length; ++i ) {
		// us this as an opportunity to zap .git, etc...
		const f = files[i] as WebkitFile;
		const rel_path = f.webkitRelativePath.substring(chop_length);
		selected_files.push({
			file: files[i],
			rel_path
		});
	}

	emit("changed", selected_files);
}


// path root inconsistent across browsers/OS:
// - chrome-mac: includes selected folder
// - chrome-win: does not -> retested: it does include it.
// - ff-win: includes selected folder
// - ff-mac: includes
// - safari-mac: includes
// - Edge/win: includes selected folder
// test: http://jsfiddle.net/o46vgasx/2/
// TODO: this really needs a proper test, but not clear how to set it up.
function getPrefix(files: FileList): string {
	let prefix = '';
	for( let i=0; i<files.length; ++i ) {
		const f = files[i] as WebkitFile;
		let wrp = f.webkitRelativePath;
		const index = wrp.indexOf('/');
		let p;
		if( index ) p = wrp.substring( 0, index );
		else p = '';

		if( i == 0 ) prefix = p;
		else if( prefix !== p ) prefix = '';
	}

	return prefix;
}

// add support for:
// - upload single file (zip)
// - drop file or directory
// - pick by url? does that belong here?

</script>
<template>
	<input type="file" name="app_dir" ref="input_elem" webkitdirectory @input="selected" />
</template>

