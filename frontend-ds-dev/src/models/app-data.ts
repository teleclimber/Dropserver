import {reactive} from 'vue';

import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

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


type AppProcessEvent = {
	processing: boolean,
	step: string
	errors: string[],
}

class AppData {

	last_processing_event :AppProcessEvent = {
		processing: true,
		step: 'waiting...',
		errors: []
	};

	name = "";
	version = "0.0.0";
	schema = 0;
	api = 0;
	migrations: MigrationStep[] = [];
    schemas: number[] = [];
	user_permissions: AppspaceUserPermission[] = [];

	_start() {
		twineClient.registerService(13, this);
	}
	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case 12:
				this.handleAppDataMessage(m);
				break;
			case 13:
				this.handleAppGetEventMessage(m);
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
		}
		catch(e) {
			m.sendError("error processing app version data "+e);
			console.error(e);
			return;
		}

		if( !this.user_permissions ) this.user_permissions = [];
	
		m.sendOK();
	}

	handleAppGetEventMessage(m:ReceivedMessageI) {
		try {
			this.last_processing_event = <AppProcessEvent>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
		}
		catch(e) {
			m.sendError("error processing app get event data "+e);
			console.error(e);
			return;
		}
		m.sendOK();
	}

	get possible_migrations() {
		if( this.schemas.length === 0 ) return [];
		const lowest = this.schemas[0];
		return [lowest-1, ...this.schemas];
	}
}

const appData = reactive(new AppData());
appData._start();
export default appData;
