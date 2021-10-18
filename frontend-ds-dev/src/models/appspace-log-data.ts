import {reactive} from 'vue';
import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

const route_commands = {
	log_event: 11,
};

export type AppspaceLogEvent = {
	appspace_id: number,
	time: Date,
	source: string,
	message: string
}

// AppspaceLogData records migration jobs and their updates.
class AppspaceLogData {
	events :AppspaceLogEvent[] = [];
	_start() {
		twineClient.registerService(15, this);
	}
	handleMessage(m:ReceivedMessageI) {
		switch(m.command){
			case route_commands.log_event:
				this.handleLogEvent(m);
			break;
			default:
				m.sendError("unrecognized service");
		}
	}
	handleLogEvent(m:ReceivedMessageI) {
		try {
			const event = <AppspaceLogEvent>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			this.events.push(event);
		}
		catch(e) {
			m.sendError("error processing appspace log event "+e);
			console.error(e);
			return;
		}

		m.sendOK();
	}
}

const appspaceLogData = reactive(new AppspaceLogData());
appspaceLogData._start();

export default appspaceLogData;