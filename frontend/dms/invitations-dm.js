import axios from 'axios';

import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

class InvitationsDM {
	constructor() {
		this.invitations = [];

		this.fetchAll();
	}

	async add( data ) {
		//this.users.push( new User( this, { email: Math.random()+'' } ) );
		const resp = await axios.post( '/api/admin/invitation', data );
		runInAction( () => {
			if( resp.data ) this.invitations.push( resp.data );
		});
	}
	async del( invitation ) {
		const resp = await axios.delete( '/api/admin/invitation/'
			+encodeURIComponent(invitation.email) );
		runInAction( () => {
			const index = this.invitations.findIndex( i => i.email === invitation.email );
			this.invitations.splice( index, 1 );
		});
	}

    async fetchAll() {
		const resp = await axios.get( '/api/admin/invitation' );
		runInAction( () => {
			this.invitations = resp.data;
		});
	}

	exists( email ) {
		return this.invitations.find( i => i.email === email );
	}
}
decorate( InvitationsDM, {
	invitations: observable,
	add: action.bound,
	//del: action.bound,
});

export default InvitationsDM;