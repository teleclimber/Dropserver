import axios from 'axios';

import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

class UsersDM {
	constructor() {
		this.users = [];

		this.fetchUsers();
	}

	addUser() {
		this.users.push( new User( this, { email: Math.random()+'' } ) );
	}
	deleteUser( del_user ) {
		const index = this.users.findIndex( u => del_user.email === u.email );
		this.users.splice( index, 1 );
	}

	dingIt( email ) {
		const index = this.users.findIndex( u => email === u.email );
		if( index === -1 ) return;
		this.users[index].email = this.users[index].email+' DING';
	}
    
    async fetchUsers() {
		const resp = await axios.get( '/api/admin/user' );
		runInAction( () => {	// required because using mobx in strict mode
			this.users = resp.data;		//resp.data.map( u => new User( this, u ) );
		});
	}
	// fetchUsers() {
	// 	const self = this;
	// 	(flow( function * () {
	// 		const resp = yield axios.get( '/api/admin/user' );
	// 		self.users = resp.data.map( u => new User( self, u ) );
	// 	}))();
	// }
	
}
decorate( UsersDM, {
	users: observable,
	addUser: action.bound,
	deleteUser: action.bound,
	dingIt: action.bound
	//fetchUsers: action.bound
});

export default UsersDM;