import { action, computed, observable, decorate, observe, runInAction, autorun } from "mobx";

interface WebkitFile extends File {
	webkitRelativePath: string;
}

export default class SelectAppFilesVM {
	@observable private _file_list: { file_list: undefined | FileList };
	@observable metadata: {
		app_name: string,
		version: string,
		schema: number,
		api: number
	} | undefined;
	@observable app_meta_error: string = '';

	constructor() {
		this._file_list = { file_list: undefined };
		observe(this, 'app_files', () => {
			this.readAppMeta();
		});
	}

	@action
	readAppMeta() {
		this.app_meta_error = '';
		if( !this.app_files ) return undefined;

		const schema_version:number = this.getSchemaVersion();

		const app_json_file = this.app_files.find( (s:SelectedFile) => s.rel_path === 'dropapp.json');

		if( !app_json_file ) {
			this.metadata = undefined;	// not sure this will set correctly
			return;
		}

		const reader = new FileReader();
		reader.readAsText(app_json_file.file, "UTF-8");
		reader.onerror = (event) => {
			runInAction( () => {
				this.metadata = undefined;
				this.app_meta_error = 'Failed to read dropapp.json';
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
					this.app_meta_error = 'Failed to parse dropapp.json';
				});
			}

			if( app_data ) {
				// should probably verify data is at least believable
				// version is properly interpreted as semver for ex
				runInAction( () => {
					this.metadata = {
						app_name: app_data.name,
						version: app_data.version,
						schema: schema_version,
						api: app_data.api
					};
					
				});
			}
		}
	}

	getSchemaVersion() : number {
		if( !this.app_files ) return 0;

		const re = /migrations\/(\d+)\//;	// do we need to use OS-specific seprators?

		let v = 0;
		this.app_files.forEach( (f:SelectedFile) => {
			const matches = f.rel_path.match(re);
			if( matches && matches.length === 2 ) {
				const fv = Number(matches[1]);
				// we should detect 0 as a bad schema version. Minimum is 1.
				// we should warn on discontinuous versions?
				if( fv > v ) v = fv;
			}
		});
		return v;
	}

	@action
	setFileList(files:FileList) {
		this._file_list = { file_list: files };	// trick to force mobx to see a new list of files, since borwser reuses FileList object
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
		return this.app_meta_error;
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