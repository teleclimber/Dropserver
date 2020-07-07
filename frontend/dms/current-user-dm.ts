import ds_axios from '../ds-axios-helper-ts';

import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

export default class CurrentUserDM {
	static injectKey = Symbol();

	@observable user: {
		email: string,
		user_id: number,
		is_admin: boolean
	} | undefined;

	constructor() {
		this.fetch();
	}	

	async fetch() {
		let resp :any;
		try {
			resp = await ds_axios.get( '/api/user' );
		}
		catch(error) {
			if( error.response.status == 401 ) window.location.href = '/login';
			else throw new Error( error );
		}

		if( !resp || !resp.data ) return;

		runInAction( () => {
			const d = resp.data;
			this.user = {
				email: d.email+"",
				user_id: Number(d.user_id),
				is_admin: !!d.is_admin
			};
		});

	}

	async changePassword(old_pw: string, new_pw: string): Promise<boolean> {
		let req = {
			new: new_pw,
			old: old_pw
		};

		let resp;
		try {
			resp = await ds_axios.patch( '/api/user/password', req, {
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


