import {reactive} from 'vue';

import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

const sandboxControlService = 19;

const setInspectCmd = 14;
const stopSandboxCmd = 15;

type SandboxStatus = {
	type: string,	//app, appspace, migration
	status: Number	// Follows host side domain.SandboxStatus 
}

class SandboxControl {
	inspect  = false;

	type = ""; // "" for off, "app", "appspace", "migration"
	status = 0;	// Follows host side domain.SandboxStatus 

	_start() {
		twineClient.registerService(sandboxControlService, this);
	}
	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case 13:
				this.handleInspectMessage(m);
				break;
			case 14:
				this.handleStatusMessage(m);
				break;
			default:
				m.sendError("command not recognized: "+m.command);
		}
	}
	handleInspectMessage(m:ReceivedMessageI) {
		try {
			if( !m.payload ) throw new Error("expected a pyalooad for set inspect");
			this.inspect = !!m.payload[0]
		}
		catch(e) {
			m.sendError("error processing set inspect "+e);
			console.error(e);
			return;
		}
		m.sendOK();
	}
	handleStatusMessage(m:ReceivedMessageI) {
		try {
			const data = <SandboxStatus>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			if( data.status != 5 ) this.type = data.type;
			else this.type = "";
			this.status = Number(data.status);
		}
		catch(e) {
			m.sendError("error processing sandbox status "+e);
			console.error(e);
			return;
		}
	
		m.sendOK();
	}
	async setInspect(inspect:boolean) {
		let buf = new ArrayBuffer(1);
		let view = new DataView(buf);
		view.setUint8(0, inspect ? 1 : 0);
		const reply = await twineClient.twine.sendBlock(sandboxControlService, setInspectCmd, new Uint8Array(buf));
		if( reply.error ) {
			throw reply.error;
		}
	}
	async stopSandbox() {
		const reply = await twineClient.twine.sendBlock(sandboxControlService, stopSandboxCmd, undefined);
		if( reply.error ) {
			throw reply.error;
		}
	}
}


const sandboxControl = reactive(new SandboxControl);
sandboxControl._start();
export default sandboxControl;