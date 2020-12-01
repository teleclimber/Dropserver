// transmit selected user aprameters (public/auth/owner, permissions) to ds-dev
// obtain and maintain app declared permissions

import {computed, reactive} from 'vue';

import twineClient from './twine-client';

import {ReceivedMessageI} from '../twine-ws/twine-common';

const userService = 17;

type User = {
	type: string,
	permissions: string[]
	display_name: string,
	proxy_id: string
}

class UserData {
	// user: User = {
	// 	type: 'owner',
	// 	permissions: [],
	// 	display_name: 'Owner',
	// 	proxy_id: 'abc'
	// }

	_start() {
		twineClient.registerService(userService, this);
	}

	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			// later accept incoming stuff like app user permissions, and appspace users
			default:
				m.sendError("command not recognized: "+m.command);
		}
	}
	async setUser(user: User) {
		const userJson = JSON.stringify(user);
		const payload = new TextEncoder().encode(userJson);

		const reply = await twineClient.twine.sendBlock(userService, 11, payload);
		if( reply.error ) {
			throw reply.error;
		}
	}
}


const userData = reactive(new UserData());
userData._start();
export default userData;