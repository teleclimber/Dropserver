import ds_axios from '../ds-axios-helper-ts';

import AppspaceDM from './appspace-dm';

import { AxiosResponse } from 'axios';
import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

export default class AppspacesDM {
	static injectKey = Symbol();

	@observable appspaces: AppspaceDM[] = [];

	constructor() {}

	async fetch() {
		let resp:any;
		try {
			resp = await ds_axios.get( '/api/appspace' );
		}
		catch(e) {
			// handle error
			console.error(e, resp);
		}

		if( !resp || !resp.data || !resp.data.appspaces ) return;	// return what?

		runInAction( () => {
			this.appspaces = resp.data.appspaces.map( (a: any) => new AppspaceDM(a) );
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

	getAppVersionAppspaces(app_id:number, version: string) :AppspaceDM[] {
		const ret:AppspaceDM[] = [];
		this.appspaces.forEach( as => {
			if( as.app_id === app_id && as.app_version === version ) ret.push(as)
		});
		return ret;
	}

	// Need help to support queries involving applications and app versions
	@computed get app_version_appspaces() : { [app_id: string]: { [app_version: string]:AppspaceDM[] }} {
		let ret : { [app_id: string]: { [app_version: string]:AppspaceDM[] }} = {};
		this.appspaces.forEach( as => {
			if( !ret[as.app_id] ) ret[as.app_id] = {};
			if( !ret[as.app_id][as.app_version] ) ret[as.app_id][as.app_version] = [];
			ret[as.app_id][as.app_version].push(as);
		});

		return ret;
	}

}
