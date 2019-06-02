
import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";


class AdminInvitationsVM {
	constructor( dm, close_cb ) {
		this.dm = dm;
		this.close_cb = close_cb;

		this.showPage();
	}
	get is() {
		return 'AdminInvitations';
	}

	showPage( page_name, ctx ) {
		if( page_name === 'new' ) this.page = new AdminInviteNewVM( this.dm, this );
		else if( page_name === 'invitation' ) this.page = new AdminUpdateInvitationVM( ctx, this.dm, this );
		else this.page = new AdminInvitationsListVM( this.dm, this );
	}

	close() {
		if( this.page && this.page.is === 'AdminInvitationsList' ) {
			if( this.close_cb ) this.close_cb();
		}
		else {
			this.showPage();	//show default page.
		}
		
	}
}
decorate( AdminInvitationsVM, {
	page: observable,
	showPage: action.bound,
	showNew: action.bound,
	close: action.bound
});

//////////////////////////////////////////////////
class AdminInvitationsListVM {
	constructor( dm, parent ) {
		this.dm = dm;
		this.parent = parent;
	}
	get is() {
		return 'AdminInvitationsList';
	}
	get invitations() {
		return this.dm.invitations.map( i => new InvitationVM(i) );
	}

	showPage( page_name, ctx ) {
		this.parent.showPage( page_name, ctx );
	}

	close() {
		if( this.parent ) this.parent.close();
	}
}
decorate( AdminInvitationsListVM, {
	showNew: action.bound,
	close: action.bound	
});

class InvitationVM {
	constructor( invitation ) {
		this.invitation = invitation;
	}
	get email() {
		return this.invitation.email;
	}
}
decorate( InvitationVM, {

});

/////////////////////////////////////////////////////
// new invitation
class AdminInviteNewVM {
	constructor( dm, parent ) {
		this.dm = dm;
		this.parent = parent;
		this.exists = false;
	}
	get is() {
		return 'AdminInviteNew';
	}

	normalizeInput( data ) {
		return {
			email: data.email.toLowerCase()
		};
	}
	inputChanged( data ) {
		data = this.normalizeInput( data );
		this.exists = this.dm.exists( data.email );
	}
	async save( data ) {
		data = this.normalizeInput( data );
		await this.dm.add( data );
		this.close();
	}
	close() {
		if( this.parent ) this.parent.close();
	}
}
decorate( AdminInvitationsVM, {
	exists: observable
});

//////////////////////////////////////////////////////
// update / delete
class AdminUpdateInvitationVM {
	constructor( invitation, dm, parent ) {
		this.invitation = invitation;
		this.dm = dm;
		this.parent = parent;
	}

	get is() {
		return 'AdminUpdateInvitation';
	}

	async del() {
		await this.dm.del( this.invitation );
		this.close();
	}

	close() {
		if( this.parent ) this.parent.close();
	}
}
decorate( AdminUpdateInvitationVM, {

});

export default AdminInvitationsVM;