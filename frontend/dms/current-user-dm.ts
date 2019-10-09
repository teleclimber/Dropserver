import ds_axios from '../ds-axios-helper-ts';

import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

export default class CurrentUserDM {
	static injectKey = Symbol();

	@observable user: User | undefined;

	constructor() {
		this.fetch();
	}	

	async fetch() {
		let resp;
		try {
			resp = await ds_axios.get( '/api/user' );
		}
		catch(error) {
			if( error.response.status == 401 ) window.location.href = '/login';
			else throw new Error( error );
		}

		if( !resp || !resp.data ) return;

		const user = <User>resp.data;

		runInAction( () => {
			this.user = user;
		});

	}

	async changePassword(old_pw: string, new_pw: string): Promise<boolean> {
		let resp;
		try {
			resp = await ds_axios.patch( '/api/user/password', {
				old: old_pw,
				new: new_pw
			}, {
				validateStatus: status => status == 200 || status == 401
			} );
		}
		catch (error) {
			console.error(error);//not sure what to do.
			throw error;
		}

		// basically, barring an unmanageable error, the result should either be
		// - OK, pw changed
		// - bad old pw. so we have to return something.

		if( resp && resp.status == 200 ) {
			return true;
		}
		else {
			return false;
		}
	}
}


