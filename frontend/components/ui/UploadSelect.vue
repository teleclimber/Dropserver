<style scoped>

</style>

<template>
	<input type="file" name="app_dir" ref="app_dir" webkitdirectory @input="dirInput" />

	<!-- add support for:
		- upload single file (zip)
		- drop file or directory
		- pick by url? does that belong here?
	-->

</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

interface WebkitFile extends File {
	webkitRelativePath: string;
}

@Observer
@Component
export default class  UploadSelect extends Vue {
	@Ref('app_dir') readonly app_dir!: HTMLInputElement;

	dirInput() {
		// In future we have to track whether user selected/dropped a directory
		// or a single file or entered a url

		// check that *something* was selected for upload / whatever

		const form_data = new FormData();
		const files = this.app_dir.files;

		if( files === null ) return;	// required to keep TS from complaining about null files

		// path root inconsistent across browsers/OS:
		// - chrome-mac: includes selected folder
		// - chrome-win: does not
		// - ff-win: includes selected folder
		// - ff-mac: includes
		// - safari-mac: includes
		// test: http://jsfiddle.net/o46vgasx/2/
		// TODO: this really needs a proper test, but not clear how to set it up.
		let prefix = '';
		for( let i=0; i<files.length; ++i ) {
			const f = <WebkitFile>files[i];
			let wrp = f.webkitRelativePath;
			const index = wrp.indexOf('/');
			let p;
			if( index ) p = wrp.substring( 0, index );
			else p = '';

			if( i == 0 ) prefix = p;
			else if( prefix !== p ) prefix = '';
		}

		console.log( 'upload path prefix: '+prefix );

		const chop_length = prefix ? prefix.length +1 : 0;

		for( let i=0; i<files.length; ++i ) {
			// us this as an opportunity to zap .git, etc...
			const f = <WebkitFile>files[i];

			const rel_path = f.webkitRelativePath.substring( chop_length );
			form_data.append( 'app_dir', files[i], rel_path );
		}

		this.$emit( 'input', form_data );
	}

};

</script>