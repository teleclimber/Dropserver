import {reactive} from 'vue';

import twineClient from './twine-client';

import {ReceivedMessageI} from '../twine-ws/twine-common';

const appspaceControlService = 12;

// Base data needs rethinking.
// - app_path and appspace_path are constants 
// - app name, version, sechema can all change as dev updates code
// - appspace schema and other metadaat can ll cahnge too as the appspace is used.

// Keeping things up to date:
// - appspace status updates schemas and status
// - Need a system to watch app files and update

type AppspaceStatus = {
	appspace_id: Number
	paused: boolean
	migrating: boolean
	appspace_schema: Number
	app_version_schema: Number
	problem: boolean
}

class BaseData {
	loaded = false;

	app_path = "/";
	appspace_path = "/";

	app_name = "";
	app_version = "0.0.0";
	app_version_schema = 0;

	paused = false;
	migrating = false;
	appspace_schema = 0;
	problem = false;

	_start() {
		this.fetchInitialData();
		twineClient.registerService(13, this);
	}

	async fetchInitialData() {
		const res = await fetch('base-data');
		if( !res.ok ) {
			throw new Error("fetch error for basic data");
		}

		try {
			const data = await res.json();
			Object.assign(this, data);
		}
		catch(error) {
			console.error(error);
		}
		this.loaded = true;
	}

	handleMessage(m:ReceivedMessageI) {
		// should really read command and act on that.
		try {
			const new_status = <AppspaceStatus>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			console.log(new_status);
			Object.assign(this, new_status);
		}
		catch(e) {
			m.sendError("error processing appspace status "+e);
			console.error(e);
			return;
		}
	
		m.sendOK();
	}
}

const baseData = reactive(new BaseData());
baseData._start();
export default baseData;

// Appspace controls:
export async function  pauseAppspace(pause:boolean) {
	const cmd = pause ? 11 : 12;
	const reply = await twineClient.twine.sendBlock(appspaceControlService, cmd, undefined);
	if( reply.error ) {
		throw reply.error;
	}
}



