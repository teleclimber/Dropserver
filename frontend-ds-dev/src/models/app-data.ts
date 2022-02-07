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

class AppData {
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
			default:
				m.sendError("command not recognized: "+m.command);
		}
		
	}
	handleAppDataMessage(m:ReceivedMessageI) {
		try {
			const new_app_data = <AppFilesMetadata>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			Object.assign(this, new_app_data);
			if( !this.schemas ) this.schemas = [];

			console.debug("new app data ", new_app_data);
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

const appData = reactive(new AppData());
appData._start();
export default appData;
