import {reactive} from 'vue';

import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

const appspaceControlService = 12;

// Base data needs rethinking.
// - app_path and appspace_path are constants 
// - app name, version, sechema can all change as dev updates code
// - appspace schema and other metadaat can ll cahnge too as the appspace is used.

// Keeping things up to date:
// - appspace status updates schemas and status
// - Need a system to watch app files and update

type AppspaceUserPermission = {
	key:         string,
	name:        string,
	description: string,
}

type MigrationStep = {
	direction: "up"|"down"
	schema: number
}

// AppFilesMetadata containes metadata that can be gleaned from
// reading the application files
type AppFilesMetadata = {
	name: string,
	version: string,
	schema: number,
	api: number,
	migrations: MigrationStep[],
	schemas: number[],
	user_permissions: AppspaceUserPermission[]
}

class BaseData {
	loaded = false;

	app_path = "/";
	appspace_path = "/";

	name = "";
	version = "0.0.0";
	schema = 0;
	api = 0;
	migrations: MigrationStep[] = [];
    schemas: number[] = [];
	user_permissions: AppspaceUserPermission[] = [];

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
		switch (m.command) {
			case 12:
				this.handleAppDataMessage(m);
				break;
			default:
				m.sendError("command not recognized: "+m.command);
		}
		
	}
	handleAppDataMessage(m:ReceivedMessageI) {
		try {
			const new_app_data = <AppFilesMetadata>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			Object.assign(this, new_app_data);
			if( !this.schemas ) this.schemas = [];

			console.debug(new_app_data);
		}
		catch(e) {
			m.sendError("error processing app version data "+e);
			console.error(e);
			return;
		}

		if( !this.user_permissions ) this.user_permissions = [];
	
		m.sendOK();
	}

	get possible_migrations() {
		if( this.schemas.length === 0 ) return [];
		const lowest = this.schemas[0];
		return [lowest-1, ...this.schemas];
	}
}

const baseData = reactive(new BaseData());
baseData._start();
export default baseData;

const appspaceCmds = {
	pause: 11,
	unpause: 12,
	migrate: 13,
	importAndMigrate: 16,
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

export class ImportAndMigrate {
	public cur_state:string = "";

	reset() {
		this.cur_state = "Import and Migrate"
	}
	start() {
		this.cur_state = "Working...";
		this._run();
	}
	async _run() {
		const sent = await twineClient.twine.send(appspaceControlService, appspaceCmds.importAndMigrate, undefined);

		for await (const m of sent.incomingMessages()) {
			switch (m.command) {
				case 11:	//cur_state
					this.cur_state = new TextDecoder('utf-8').decode(m.payload);
					m.sendOK();
					break;
			
				default:
					m.sendError("What is this command?");
					throw new Error("what is this command? "+m.command);
			}
		}

		const reply = await sent.waitReply();
		if( reply.error != undefined ) {
			throw reply.error;	// TODO investigate: I think we changed error to be a string so you can throw it from the right place.
		}

		this.cur_state = 'All done!';

		setTimeout(() => {
			this.reset();
		}, 1000);
		
	}
}


