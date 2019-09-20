import ds_axios from '../ds-axios-helper.js';

import { AxiosResponse, AxiosPromise } from 'axios';
import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

export default class ApplicationsDM {
	static injectKey = Symbol();

	@observable appspaces: AppspaceMeta[] = [];

	constructor() {
		this.fetch();
	}

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

		let appspaces = <AppspaceMeta[]>resp.data.appspaces;

		runInAction( () => {
			this.appspaces = appspaces;
		});
	}

	getAppspace(appspace_id: number) {
		const a = this.appspaces.find( (a:AppspaceMeta) => a.appspace_id === appspace_id );
		if( !a ) throw new Error('appspace not found');
		return a;
	}

	async create( app_id: number, version: string ): Promise<AppspaceMeta> {

		let resp: AxiosResponse<any>;

		try {
			resp = await ds_axios.post( '/api/appspace', { app_id, version } );
		}
		catch(e) {
			// should not be an error on this?
			// or maybe something about unavailable address?
			throw(e);
		}

		//if( !resp ) return;	// shouldn't we return a appspacemeta?

		const appspace = <AppspaceMeta>resp.data.appspace;
		runInAction( () => {
			this.appspaces.push(appspace);
		});

		return appspace;
	}

}
