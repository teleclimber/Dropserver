import ds_axios from '../ds-axios-helper.js'
import { compare as semverCompare } from 'semver';

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
	// async getApplicationP(app_id: number): Promise<ApplicationMeta> {
	// 	return new Promise( (resolve) => {
	// 		const disposer = observe( this, 'fetched', change => {
	// 			if( change.newValue ) {
	// 				console.log('applications fetched!');
	// 				const a = this.applications.find( (a:ApplicationMeta) => a.app_id === app_id );
	// 				if( !a ) throw new Error('application not found');//reject?
	// 				disposer();
	// 				resolve(a);
	// 			}
	// 		});
	// 	});
	// }
	// ^^ OK problem here is that we are sometimes trying to get this before data has loaded.
	// How to deal?
	// - block everything until some data is loaded (lousy)
	// - return a promise or async (does that work in constructors?)
	// - return dummy data (perhaps with a "_loading:true" prop), and set actual data when it gets set?
	// I like the last one as it is the most reactive-friendly, but worried it might trigger other errors?
	// Because the code expects the actual data to match expectations.
	// ..like it'll throw if it tries to fetch a version and can't find it for example.


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
		apps.forEach( (a: ApplicationMeta) => sortVersions(a.versions) );

		runInAction( () => {
			this.applications = apps;
			this.fetched = true;
		});
	}

	// we could probably have a single upload function 
	async uploadNewApplication(form_data: any): Promise<UploadApplicationResp> {
		let ret: UploadApplicationResp = {error: false};
		let resp: AxiosResponse<any>;

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

	async uploadNewVersion(app_id: number, form_data: any): Promise<UploadVersionResp> {
		let ret: UploadVersionResp= {error: false};
		let resp: AxiosResponse<any>;

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

			const new_version_meta = resp.data.app_meta;

			const application = this.applications.find( (a:ApplicationMeta) => a.app_id === app_id );
			if( application == undefined ) {
				// that's an error;
				ret.error = true;
				return ret;
			}
			
			runInAction( () => {
				application.versions.push(new_version_meta);
			});
			sortVersions(application.versions);

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

function sortVersions( versions: VersionMeta[] ) {
	versions.sort( (a, b) => {
		return semverCompare(b.version, a.version);	// reverse order
	});
}