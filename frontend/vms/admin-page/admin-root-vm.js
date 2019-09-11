
import { 
	action,
	computed,
	observable,
	decorate,
	configure,
	runInAction
 } from "mobx";

configure( {enforceActions: 'always'} );

import UsersDM from '../../dms/users-dm.js';
import InstanceSettingsDM from '../../dms/instance-settings-dm.js';
import InvitationsDM from '../../dms/invitations-dm.js';

import AdminSettingsVM from './admin-settings-vm.js';
import AdminInvitationsVM from './admin-invitations-vm.js';

class AdminVM {
	constructor() {

		this.cur_modal = null;

		this.users_dm = new UsersDM();
		this.users_dm.fetchUsers();

		this.instance_settings_dm = new InstanceSettingsDM;
		this.instance_settings_dm.fetchData();

		this.invitations_dm = new InvitationsDM;
		this.invitations_dm.fetchAll();

	}

	// modals:
	closeModal() {
		this.cur_modal = null;
	}
	// settings:
	showSettings() {
		this.cur_modal = new AdminSettingsVM( this.instance_settings_dm, this.closeModal );	//this.closeModal is action.bound, so `this` will be correct.
	}
	showInvitations() {
		//check if there is a modal open currently?
		this.cur_modal = new AdminInvitationsVM( this.invitations_dm, this.closeModal );
	}


	// instance settings: 
	get registration() {
		if( !this.instance_settings_dm.data ) return '...';
		if( this.instance_settings_dm.data.registration_open === 'open' ) return 'Registration is open to the public';
		else return 'User registration is invitation only';
	}

	get num_invitations() {
		return this.invitations_dm.invitations.length;
	}
	// users:
	get users() {
		return this.users_dm.users.map( u => new User(this.users_dm,u) );
	}
}
decorate( AdminVM, {
	cur_modal: observable,
	closeModal: action.bound,
	showSettings: action.bound,
	showInvitations: action.bound

	//users: computed,	//!!! this messes up the computed. The dfault for get must be something else.

});

class User {
	constructor( users_dm, user_data ) {
		this.users_dm = users_dm;
		this.setUserData( user_data );
	}

	setUserData( user_data ) {
		this.email = user_data.email;
	}

	del() {
		//this.owner.deleteUser( this );
	}
}
decorate( User, {
	setUserData: action.bound,
	email: observable,
	//del: action.bound
});

export default AdminVM;