// reflect appspace users from ds-dev
// show display name, proxy id and permissions
// create new user, change users
// set/reflect status of is owner, which is from the appspace contact db table
// 

import {computed, reactive} from 'vue';

import twineClient from './twine-client';

import {ReceivedMessageI} from '../twine-ws/twine-common';

const userService = 17;

// local commands:
const loadAllUsers = 11;
const loadOwner = 12;

// remote commands
const userCreateCmd      = 11
const userUpdateCmd      = 12
const userDeleteCmd      = 13
const userSelectUserCmd  = 15

export type User = {
	permissions: string[],
	display_name: string,
	proxy_id: string
}

class UserData {
	users : User[] = [];
	user_proxy_id  : string = "";
	owner_proxy_id : string = "";

	_start() {
		twineClient.registerService(userService, this);
	}

	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case loadAllUsers:
				this.handleLoadAllUsers(m);
				break;
			// case loadOwner:
			// 	this.handleLoadOwner(m);
			// 	break;

			// later accept incoming stuff like app user permissions, and appspace users
			default:
				m.sendError("command not recognized: "+m.command);
		}
	}
	handleLoadAllUsers(m:ReceivedMessageI) {
		const users = JSON.parse(new TextDecoder().decode(m.payload));
		console.log(users)	// apparently users is an array []? didn't know you could do that in json.
		this.users = users.map(userFromRaw);
		m.sendOK();
	}

	getUser(proxy_id:string) : User|undefined {
		return this.users.find((u:User) => u.proxy_id === proxy_id);
	}

	isUser(proxy_id:string) :boolean {
		return proxy_id === this.user_proxy_id;
	}
	// async setUser(user: User) {
	// 	const userJson = JSON.stringify(user);
	// 	const payload = new TextEncoder().encode(userJson);

	// 	const reply = await twineClient.twine.sendBlock(userService, 11, payload);
	// 	if( reply.error ) {
	// 		throw reply.error;
	// 	}
	// }

	async addUser(display_name: string, permissions: string[]) {
		const user :User = {
			display_name,
			permissions,
			proxy_id: ""
		};

		const payload = new TextEncoder().encode(JSON.stringify(user))
		const reply = await twineClient.twine.sendBlock(userService, userCreateCmd, payload);

		if( reply.error ) {
			console.error(reply.error);
			return;
		}

		const u = JSON.parse(new TextDecoder().decode(reply.payload))
		this.users.push(userFromRaw(u));

	}

	async editUser(proxy_id:string, display_name: string, permissions: string[]) {
		const user :User = {
			proxy_id,
			display_name,
			permissions
		};

		const payload = new TextEncoder().encode(JSON.stringify(user))
		const reply = await twineClient.twine.sendBlock(userService, userUpdateCmd, payload);

		if( !reply.ok ) {
			console.error(reply.error);
			return;
		}

		const i = this.users.findIndex((u:User) => u.proxy_id === proxy_id);
		if( i == -1 ) throw new Error("couldn't find user to update");
		this.users[i] = user;
	}
	async deleteUser(proxy_id:string) {
		const payload = new TextEncoder().encode(proxy_id);
		const reply = await twineClient.twine.sendBlock(userService, userDeleteCmd, payload);

		if( !reply.ok ) {
			console.error(reply.error);
		}

		const i = this.users.findIndex((u:User) => u.proxy_id === proxy_id);
		if( i == -1 ) throw new Error("couldn't find user to update");
		this.users.splice(i, 1);

		if( this.owner_proxy_id === proxy_id ) this.owner_proxy_id = "";
		if( this.user_proxy_id === proxy_id ) this.user_proxy_id = "";
	}

	async setUser(proxy_id :string) {
		const payload = new TextEncoder().encode(proxy_id);
		const reply = await twineClient.twine.sendBlock(userService, userSelectUserCmd, payload);

		if( !reply.ok ) {
			console.error(reply.error);
		}
		this.user_proxy_id = proxy_id;
	}
}

function userFromRaw(u:any) :User {
	return {
		proxy_id: u.proxy_id+'',
		display_name: u.display_name+'',
		permissions: u.permissions
	}
}


const userData = reactive(new UserData());
userData._start();
export default userData;