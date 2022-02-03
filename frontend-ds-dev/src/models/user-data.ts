import {computed, reactive} from 'vue';

import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

const userService = 17;

// local commands:
const loadAllUsers = 11;
const setCurrentUser = 12;

// remote commands
const userCreateCmd      = 11
const userUpdateCmd      = 12
const userDeleteCmd      = 13
const userSelectUserCmd  = 15

export type User = {
	permissions: string[],
	display_name: string,
	avatar: string,
	proxy_id: string
}

class UserData {
	users : User[] = [];
	user_proxy_id  : string = "";

	_start() {
		twineClient.registerService(userService, this);
	}

	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case loadAllUsers:
				this.handleLoadAllUsers(m);
				break;
			case setCurrentUser:
				this.handleSetCurrentUser(m);
				break;
			// later accept incoming stuff like app user permissions
			default:
				m.sendError("command not recognized: "+m.command);
		}
	}
	handleLoadAllUsers(m:ReceivedMessageI) {
		const payload = new TextDecoder().decode(m.payload);
		console.log( "users payload", payload);
		const users = JSON.parse(payload);
		console.log(users)	// apparently users is an array []? didn't know you could do that in json.
		this.users = users.map(userFromRaw);
		m.sendOK();
	}
	handleSetCurrentUser(m:ReceivedMessageI) {
		this.user_proxy_id = new TextDecoder().decode(m.payload);
		m.sendOK();
	}

	getUser(proxy_id:string) : User|undefined {
		return this.users.find((u:User) => u.proxy_id === proxy_id);
	}
	getActiveUser() :User|undefined {
		return this.getUser(this.user_proxy_id);
	}
	isUser(proxy_id:string) :boolean {
		return proxy_id === this.user_proxy_id;
	}

	async addUser(display_name: string, avatar:string, permissions: string[]) {
		const user :User = {
			display_name,
			avatar,
			permissions,
			proxy_id: ""
		};

		const payload = new TextEncoder().encode(JSON.stringify(user))
		const reply = await twineClient.twine.sendBlock(userService, userCreateCmd, payload);

		if( reply.error ) {
			console.error(reply.error);
			return;
		}
	}

	async editUser(proxy_id:string, display_name: string, avatar:string, permissions: string[]) {
		const user :User = {
			proxy_id,
			display_name,
			avatar,
			permissions
		};

		const payload = new TextEncoder().encode(JSON.stringify(user))
		const reply = await twineClient.twine.sendBlock(userService, userUpdateCmd, payload);

		if( reply.error ) {
			console.error(reply.error);
			return;
		}
	}
	async deleteUser(proxy_id:string) {
		const payload = new TextEncoder().encode(proxy_id);
		const reply = await twineClient.twine.sendBlock(userService, userDeleteCmd, payload);

		if( !reply.ok ) {
			console.error(reply.error);
		}
	}

	async setActiveUser(proxy_id :string) {
		const payload = new TextEncoder().encode(proxy_id);
		const reply = await twineClient.twine.sendBlock(userService, userSelectUserCmd, payload);

		if( !reply.ok ) {
			console.error(reply.error);
		}
	}
}

function userFromRaw(u:any) :User {
	return {
		proxy_id: u.proxy_id+'',
		display_name: u.display_name+'',
		avatar: u.avatar+'',
		permissions: u.permissions
	}
}


const userData = reactive(new UserData());
userData._start();
export default userData;