import { action, computed, observable, decorate, observe, runInAction, autorun } from "mobx";

interface WebkitFile extends File {
	webkitRelativePath: string;
}

export default class SelectAppFilesVM {
	@observable private _file_list: { file_list: undefined | FileList };
	@observable metadata: {
		app_name: string,
		version: string,
		schema: number
	} | undefined;
	@observable app_json_error: string = '';

	constructor() {
		this._file_list = { file_list: undefined };
		observe(this, 'app_json_file', () => {
			this.readAppJson();
		});
	}

	@computed get app_json_file(): SelectedFile | undefined {
		if( !this.app_files ) return undefined;
		return this.app_files.find( (s:SelectedFile) => s.rel_path === 'application.json');
	}

	readAppJson() {
		if( !this.app_json_file ) {
			this.metadata = undefined;	// not sure this will set correctly
			return;
		}

		const reader = new FileReader();
		reader.readAsText(this.app_json_file.file, "UTF-8");
		reader.onerror = (event) => {
			runInAction( () => {
				this.metadata = undefined;
				this.app_json_error = 'Failed to read application.json';
			});
		}
		reader.onload = () => {
			let app_data:any;
			try {
				app_data = JSON.parse(<string>reader.result);
			}
			catch(e) {
				runInAction( () => {
					this.metadata = undefined;
					this.app_json_error = 'Failed to parse application.json';
				});
			}

			if( app_data ) {
				// should probably verify data is at least believable
				// version is properly interpreted as semver for ex
				// schema is a number.
				runInAction( () => {
					this.metadata = {
						app_name: app_data.name,
						version: app_data.version,
						schema: app_data.schema ? Number(app_data.schema) : 0
					};
					this.app_json_error = '';
				});
			}
		}
	}

	@action
	setFileList(files:FileList) {
		this._file_list = { file_list: files };
	}

	//@computed //precisely not computed!
	get file_list() : FileList | undefined {
		return this._file_list.file_list;
	}

	@computed get app_files(): SelectedFile[] | undefined {
		if( !this.file_list ) return;
		// should potentially reset error and metadata and files...

		const files = this.file_list;

		const prefix = getPrefix(files as FileList);
		const chop_length = prefix ? prefix.length +1 : 0;

		const selected_files: SelectedFile[] = [];

		for( let i=0; i<files.length; ++i ) {
			// us this as an opportunity to zap .git, etc...
			const f = <WebkitFile>files[i];
			const rel_path = f.webkitRelativePath.substring(chop_length);
			selected_files.push({
				file: files[i],
				rel_path
			});
		}

		return selected_files;
	}

	@computed get error(): string {
		if( !this.app_files ) return '';	// no error if nothing selected
		if( !this.app_json_file ) return 'Failed to find application.json';
		return this.app_json_error;
	}
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
		const f = <WebkitFile>files[i];
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