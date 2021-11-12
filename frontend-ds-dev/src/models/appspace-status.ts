
import {reactive} from 'vue';

import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

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