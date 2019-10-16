// augmented appspace
import ds_axios from '../ds-axios-helper-ts';

import ApplicationsDM from './applications-dm';
import ApplicationDM from './application-dm';

import { AxiosResponse, AxiosPromise } from 'axios';
import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import { AppspaceMeta, VersionMeta } from '../generated-types/userroutes-classes';
import { PostAppspacePauseReq } from '../generated-types/userroutes-interfaces';

import autoDecorate from '../utils/mobx-auto-decorate';
autoDecorate(AppspaceMeta);

export default class AppspaceDM extends AppspaceMeta {	// maybe extend generated class??


	constructor(public appspace: AppspaceMeta) {
		super(appspace);
	}

	async doPause(pause:boolean) {
		let resp: AxiosResponse<any>;
		const req_data : PostAppspacePauseReq = { pause	};

		try {
			resp = await ds_axios.post( '/api/appspace/'+this.appspace_id+'/pause', req_data );	// this should be an interface?
		}
		catch(e) {
			// should not be an error on this?
			// or maybe something about unavailable address?
			throw(e);
		}

		if( resp.status == 200 ) {
			runInAction( () => {
				//this.appspace.paused = pause;
				this.paused = pause;
			});
		}
	}

	// chang version -> server
}