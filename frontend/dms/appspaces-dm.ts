import ds_axios from '../ds-axios-helper-ts';

import AppspaceDM from './appspace-dm';
import ApplicationsDM from './applications-dm';

import { AxiosResponse, AxiosPromise } from 'axios';
import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

export default class AppspacesDM {
	static injectKey = Symbol();

	@observable appspaces: AppspaceDM[] = [];

	constructor() {}

	async fetch() {
		let resp;
		try {
			resp = await ds_axios.get( '/api/appspace' );
		}
		catch(e) {
			// handle error
			console.error(e, resp);
		}

		if( !resp || !resp.data || !resp.data.appspaces ) return;	// return what?

		let appspaces = <AppspaceMeta[]>resp.data.appspaces;// TODO: fix it up?

		runInAction( () => {
			this.appspaces = appspaces.map( (a: any) => new AppspaceDM(a) );
		});
	}

	getAppspace(appspace_id: number) : AppspaceDM {
		const a = this.appspaces.find( (a:AppspaceDM) => a.appspace_id === appspace_id );
		if( !a ) throw new Error('appspace not found');
		return a;
	}

	async create( app_id: number, version: string ): Promise<{appspace:AppspaceDM, job_id: number}> {

		let resp: AxiosResponse<any>;

		try {
			resp = await ds_axios.post( '/api/appspace', { app_id, version } );
		}
		catch(e) {
			// should not be an error on this?
			// or maybe something about unavailable address?
			throw(e);
		}

		const appspace = new AppspaceDM(resp.data.appspace);
		runInAction( () => {
			this.appspaces.push(appspace);
		});

		const job_id = Number(resp.data.job_id);

		return {appspace, job_id};
	}

}
