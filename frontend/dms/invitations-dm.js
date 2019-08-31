import ds_axios from '../ds-axios-helper.js'

import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

class InvitationsDM {
	constructor() {
		this.invitations = [];
	}

	async add( data ) {
		let req = {
			user_invitation: {
				email: data.email
			}
		};
		ds_axios.post( '/api/admin/invitation', req ).then( resp => {
			this.fetchAll();
		}).catch( e => {
			console.error(e);
		});
		
	}
	async del( invitation ) {
		ds_axios.delete( '/api/admin/invitation/'+encodeURIComponent(invitation.email) ).then( () => {
			runInAction( () => {
				const index = this.invitations.findIndex( i => i.email === invitation.email );
				this.invitations.splice( index, 1 );	// can we splice in mobx? yes, apparently we can.
			});
		});
	}

    async fetchAll() {
		ds_axios.get( '/api/admin/invitation' ).then(resp => {
			runInAction( () => {
				this.invitations = resp.data.user_invitations || [];
			});
		}).catch( e => {
			console.error(e);
		});
		
	}

	exists( email ) {
		return this.invitations.find( i => i.email === email );
	}
}
decorate( InvitationsDM, {
	invitations: observable
});

export default InvitationsDM;