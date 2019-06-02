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

<script>
export default {
	name: 'UploadSelect',
	methods: {
		dirInput: function() {
			// In future we have to track whether user selected/dropped a directory
			// or a single file or entered a url

			// check that *something* was selected for upload / whatever

			const form_data = new FormData();
			const files = this.$refs.app_dir.files;

			// path root inconsistent across browsers/OS:
			// - chrome-mac: includes selected folder
			// - chrome-win: does not
			// - ff-win: includes selected folder
			// - ff-mac: includes
			// - safari-mac: includes
			// test: http://jsfiddle.net/o46vgasx/2/
			let prefix = false;
			for( let i=0; i<files.length; ++i ) {
				let wrp = files[i].webkitRelativePath;
				const index = wrp.indexOf('/');
				let p;
				if( index ) p = wrp.substring( 0, index );
				else p = '';

				if( prefix === false ) prefix = p;
				else if( prefix !== p ) prefix = '';
			}

			console.log( 'upload path prefix: '+prefix );

			const chop_length = prefix ? prefix.length +1 : 0;

			for( let i=0; i<files.length; ++i ) {
				// us this as an opportunity to zap .git, etc...

				const rel_path = files[i].webkitRelativePath.substring( chop_length );
				form_data.append( 'app_dir', files[i], rel_path );
			}

			//this.vm.createDoNext( form_data );
			this.$emit( 'input', form_data );
		},
	}

};

</script>