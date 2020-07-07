// augmented appspace
import ds_axios from '../ds-axios-helper-ts';

import {VersionDM} from './application-dm';

import { AxiosResponse, AxiosPromise } from 'axios';
import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

export default class AppspaceDM {
	@observable appspace_id: number;
	@observable app_id: number;
	@observable app_version: string;
	@observable subdomain: string;
	@observable created_dt: Date;
	@observable paused: boolean;

	constructor(data: any) {
		this.appspace_id = Number(data.appspace_id);
		this.app_id = Number(data.app_id);
		this.app_version = data.app_version+"";
		this.subdomain = data.subdomain+"";
		this.created_dt = new Date(data.created_dt);
		this.paused = !!data.paused;
	}

	async doPause(pause:boolean) {
		let resp: AxiosResponse<any>;
		const req_data = { pause };

		try {
			resp = await ds_axios.post( '/api/appspace/'+this.appspace_id+'/pause', req_data );
		}
		catch(e) {
			// should not be an error on this?
			// or maybe something about unavailable address?
			throw(e);
		}

		if( resp.status == 200 ) {
			runInAction( () => {
				this.paused = pause;
			});
		}
	}

	async changeVersion(ver: VersionDM) {
		let req_data = { version: ver.version };
		let resp: AxiosResponse<any>;
		
		try {
			resp = await ds_axios.post( '/api/appspace/'+this.appspace_id+'/version', req_data );
		}
		catch(e) {
			// should not be an error on this?
			// or maybe something about unavailable address?
			throw(e);
		}

		if( resp.status == 200 ) {	// will we really get a 200?
			runInAction( () => {
				this.app_version = req_data.version;
			});
		}
	}
}