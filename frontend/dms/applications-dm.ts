import ds_axios from '../ds-axios-helper-ts';
import { compare as semverCompare, gt as semverGt, lt as semverLt } from 'semver';

import { action, computed, observable, decorate, configure, runInAction, flow, observe } from "mobx";
import { AxiosResponse } from 'axios';

import { GetAppsResp, PostAppResp, PostVersionResp, ApplicationMeta, VersionMeta } from '../generated-types/userroutes-classes';

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

	constructor() {}

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
		// it's the first one that is less than passed version
		const versions = this.getApplication(app_id).versions;
		return versions.find( (v:VersionMeta) => semverLt(v.version, version));
	}
	getNextVersion(app_id: number, version:string): VersionMeta | undefined {
		const versions = this.getApplication(app_id).versions;
		return versions.slice().reverse().find( (v:VersionMeta) => semverGt(v.version, version));
	}

	async fetchAll() {
		let resp;
		try {
			resp = await ds_axios.get( '/api/application' );
		}
		catch(e) {
			// handle error
			console.error(e, resp);
		}

		if( !resp || !resp.data || !resp.data.apps ) return;	// return what?

		let apps_resp = new GetAppsResp(resp.data);
		apps_resp.apps.forEach( (a: ApplicationMeta) => a.versions = sortVersions(a.versions) );

		runInAction( () => {
			this.applications = apps_resp.apps;
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
			const resp_inst = new PostAppResp(resp.data);
			ret.app_meta = resp_inst.app_meta;
			// TODO: need to sort versions, (but anyhoo better to have app as separate dm that handles taht auto.)
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

			const resp_inst = new PostVersionResp(resp.data);

			runInAction( () => {
				application.versions.push(resp_inst.version_meta);
				application.versions = sortVersions(application.versions);
			});
			
			ret.version_meta = resp_inst.version_meta;
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

		const application = this.getApplication(app_id);
		if( !application ) return;	//error	// will throw before we get here anywyas.

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

// Versions are sorted in reverse order. 
// -> the latest version is at versions[0]
function sortVersions( versions: VersionMeta[] ) :VersionMeta[] {
	const v = versions.slice();
	v.sort( (a, b) => {
		return semverCompare(b.version, a.version);	// reverse order
	});
	return v;
}