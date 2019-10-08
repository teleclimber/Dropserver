import ds_axios from '../ds-axios-helper.js'
import { compare as semverCompare, gt as semverGt } from 'semver';

import { action, computed, observable, decorate, configure, runInAction, flow, observe } from "mobx";
import { AxiosResponse } from 'axios';

type UploadApplicationResp = {
	error: boolean,
	error_message?: string,
	app_meta?: ApplicationMeta
}
type UploadVersionResp = {
	error: boolean,
	error_message?: string,
	version_meta?: VersionMeta
}

export default class ApplicationsDM {
	static injectKey = Symbol();

	@observable applications: ApplicationMeta[] = [];

	@observable fetched = false;

	constructor() {
		this.fetch();
	}

	getApplication(app_id: number) : ApplicationMeta {
		const a = this.applications.find( (a:ApplicationMeta) => a.app_id === app_id );
		if(!a) throw new Error('application not found');
		return a;
	}
	getVersion(app_id: number, version: string): VersionMeta {
		const a = this.getApplication(app_id);
		const v = a.versions.find( (v:VersionMeta) => v.version === version );
		if(!v) throw new Error('version not found');
		return v
	}
	versionExists(app_id: number, version: string): boolean {
		const a = this.getApplication(app_id);
		const v = a.versions.find( (v:VersionMeta) => v.version === version );
		return !!v;
	}
	getPrevVersion(app_id: number, version:string): VersionMeta | undefined {
		const versions = this.getApplication(app_id).versions;
		if( versions.length === 0 ) return;
		if( versions.length === 1 && semverGt(version, versions[0].version) ) return versions[0];
		const i = versions.findIndex( (v:VersionMeta) => semverGt(v.version, version) );
		if( i > 0 ) return versions[i-1];
		return undefined;
	}
	getNextVersion(app_id: number, version:string): VersionMeta | undefined {
		const versions = this.getApplication(app_id).versions;
		return versions.find( (v:VersionMeta) => semverGt(v.version, version));
	}

	async fetch() {
		let resp;
		try {
			resp = await ds_axios.get( '/api/application' );
		}
		catch(e) {
			// handle error
			console.error(e, resp);
		}

		if( !resp || !resp.data || !resp.data.apps ) return;	// return what?

		let apps = <ApplicationMeta[]>resp.data.apps;
		apps.forEach( (a: ApplicationMeta) => a.versions = sortVersions(a.versions) );

		runInAction( () => {
			this.applications = apps;
			this.fetched = true;
		});
	}

	// we could probably have a single upload function 
	async uploadNewApplication(selected_files: SelectedFile[]): Promise<UploadApplicationResp> {
		let ret: UploadApplicationResp = {error: false};
		let resp: AxiosResponse<any>;

		const form_data = new FormData();
		selected_files.forEach((sf)=> {
			form_data.append( 'app_dir', sf.file, sf.rel_path );
		});

		try {
			resp = await ds_axios.post( '/api/application/', form_data, {	
				headers: {
					'Content-Type': 'multipart/form-data'
				},
				validateStatus: status => status == 200 || status == 422
			});
		}
		catch(e) {
			// handle it.
			ret.error = true;
			return ret;
		}

		if( resp.status == 422 ) {
			ret.error = true;
			ret.error_message = resp.data.error;
		}
		else {
			ret.error = false;
			ret.app_meta = resp.data.app_meta;
			runInAction( () => {
				this.applications.push(resp.data.app_meta);
			});
		}

		return ret;
	}

	async uploadNewVersion(app_id: number, selected_files: SelectedFile[]): Promise<UploadVersionResp> {
		let ret: UploadVersionResp= {error: false};
		let resp: AxiosResponse<any>;

		const application = this.getApplication(app_id);

		const form_data = new FormData();
		selected_files.forEach((sf)=> {
			form_data.append( 'app_dir', sf.file, sf.rel_path );
		});

		try {
			resp = await ds_axios.post( '/api/application/'+encodeURIComponent(app_id)+'/version/', form_data, {	
				headers: {
					'Content-Type': 'multipart/form-data'
				},
				validateStatus: status => status == 200 || status == 422
			});
		}
		catch(e) {
			ret.error = true;
			return ret;
		}

		if( resp.status == 422 ) {
			ret.error = true;
			ret.error_message = resp.data.error;
		}
		else {
			ret.error = false;

			const new_version_meta = resp.data.version_meta;
			
			runInAction( () => {
				application.versions.push(new_version_meta);
				application.versions = sortVersions(application.versions);
			});
			
			ret.version_meta = new_version_meta;
		}

		return ret;
	}

	async deleteVersion( app_id: number, version:string ) {
		try {
			await ds_axios.delete( '/api/application/'
				+encodeURIComponent(app_id)+'/version/'+encodeURIComponent(version) );
		}
		catch(e) {
			// blah blah
			return;
		}

		const application = this.applications.find( (a:ApplicationMeta) => a.app_id === app_id );
		if( !application ) return;	//error

		const i = application.versions.findIndex( (v: VersionMeta) => v.version === version );
		runInAction( () => {
			application.versions.splice( i, 1 );
		});
	}
	deleteApplication( app_id: number ) {
		try {
			ds_axios.delete( '/api/application/'+encodeURIComponent(app_id) );
		}
		catch(e) {
			return;	//error
		}
		
		const index = this.applications.findIndex( (a:ApplicationMeta) => a.app_id === app_id );
		runInAction( () => {
			this.applications.splice( index, 1 );
		});
	}
}

function sortVersions( versions: VersionMeta[] ) :VersionMeta[] {
	const v = versions.slice();
	v.sort( (a, b) => {
		return semverCompare(b.version, a.version);	// reverse order
	});
	return v;
}