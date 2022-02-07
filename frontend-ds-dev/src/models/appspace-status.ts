
import {reactive} from 'vue';

import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

const appspaceControlService = 12;
const appspaceStatusService = 18;

type AppspaceStatusEvent = {
	appspace_id: Number
	paused: boolean
	temp_paused: boolean
	temp_pause_reason: string
	appspace_schema: Number
	app_version_schema: Number
	problem: boolean
}

class AppspaceStatus {
	paused = false;
	temp_paused = false;
	temp_pause_reason = '';
	appspace_schema = 0;
	app_version_schema = 0;
	problem = false;

	loaded = false;

	_start() {
		twineClient.registerService(appspaceStatusService, this);
	}
	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case 11:
				this.setStatus(m);
				break;
			default:
				m.sendError("command not recognized: "+m.command);
		}
	}
	setStatus(m:ReceivedMessageI) {
		try {
			const new_status = <AppspaceStatusEvent>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			Object.assign(this, new_status);
			this.loaded = true;
		}
		catch(e) {
			m.sendError("error processing appspace status "+e);
			console.error(e);
			return;
		}
	
		m.sendOK();
	}
}

const appspaceStatus = reactive(new AppspaceStatus);
appspaceStatus._start();
export default appspaceStatus;


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
		this.cur_state = "Reload Appspace"
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
