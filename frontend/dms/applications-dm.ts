import ds_axios from '../ds-axios-helper-ts';

import { action, computed, observable, decorate, configure, runInAction, flow, observe } from "mobx";
import { AxiosResponse } from 'axios';

import ApplicationDM from './application-dm';

import { GetAppsResp, PostAppResp, ApplicationMeta, VersionMeta } from '../generated-types/userroutes-classes';
// GetAppsResp no longer used. Still in flux what/how to use generated types and classes.

type UploadApplicationResp = {
	error: boolean,
	error_message?: string,
	application?: ApplicationDM
}


export default class ApplicationsDM {
	static injectKey = Symbol();

	@observable applications: ApplicationDM[] = [];

	@observable fetched = false;

	constructor() {}

	getApplication(app_id: number) : ApplicationDM {
		const a = this.applications.find( (a:ApplicationDM) => a.app_id === app_id );
		if(!a) throw new Error('application not found');
		return a;
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

		//let apps_resp = new GetAppsResp(resp.data);	
		//^^ actually not using the GetAPpsResp generated type because it complicates creation of extended classes that have that data.
		let apps = <ApplicationMeta[]>resp.data.apps;

		runInAction( () => {
			this.applications = apps.map( (a: any) => new ApplicationDM(a) );
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

			ret.application = new ApplicationDM(resp.data.app_meta);
			runInAction( () => {
				if( !ret.application ) throw new Error('constructor returned undefined');	// friggin typescript
				this.applications.push(ret.application);
			});
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
		
		const index = this.applications.findIndex( (a:ApplicationDM) => a.app_id === app_id );
		runInAction( () => {
			this.applications.splice( index, 1 );
		});
	}
}
