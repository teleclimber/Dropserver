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
	app_migrations :number[] = [];

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

	// TODO: this very badly needs testing!
	get possible_migrations() {
		const ret :number[] = [];
		const cur_schema = this.appspace_schema;
		let cur_i = this.app_migrations.indexOf(cur_schema);
		if( cur_i === -1 ) return ret;

		let i = 0;
		while(true) {
			++i;
			if(this.app_migrations[cur_i + i] === cur_schema+i ) ret.push(cur_schema+i);
			else break;
		}

		i = 0;
		while(cur_i + i > 0) {
			--i;
			if(this.app_migrations[cur_i + i] === cur_schema+i ) ret.unshift(cur_schema+i);
			else break;
		}

		return ret;
	}
}

const baseData = reactive(new BaseData());
baseData._start();
export default baseData;

const appspaceCmds = {
	pause: 11,
	unpause: 12,
	migrate: 13
}

// Appspace controls:
export async function  pauseAppspace(pause:boolean) {
	const cmd = pause ? appspaceCmds.pause : appspaceCmds.unpause;
	const reply = await twineClient.twine.sendBlock(appspaceControlService, cmd, undefined);
	if( reply.error ) {
		throw reply.error;
	}
}

export async function  runMigration(to_schema:number) {
	// check if to schema is legit.
	// - it should not be the current appspace schema
	// - it should be a schema that is included in the app's migrations dir, along with every other migration level.
	// Other option is to baseData produce a list of migration levels that are legit
	// use that in the drop-down, and that's that.

	let buf = new ArrayBuffer(2);
	let view = new DataView(buf);
	view.setUint16(0, to_schema);
	
	const reply = await twineClient.twine.sendBlock(appspaceControlService, appspaceCmds.migrate, new Uint8Array(buf));
	if( reply.error ) {
		throw reply.error;
	}
}



