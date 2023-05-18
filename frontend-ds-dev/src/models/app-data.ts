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

type AppManifest = {
	name :string,
	short_description: string,
	version :string,
	release_date: Date|undefined,
	main: string,	// do we care here?
	schema: number,
	migrations: MigrationStep[],
	lib_version: string,	//semver
	signature: string,	//later
	code_state: string,	 // ? later
	icon: string,	// how to reference icon? app version should have  adefault path so no need to reference it here? Except to know if there is one or not
	
	authors: string[],	// later, 
	description: string,	// actually a reference to a long description. Later.
	release_notes: string,	// ref to a file or something...
	code: string,	// URL to code repo. OK.
	homepage: string,	//URL to home page for app
	help: string,	// URL to help
	license: string,	// SPDX format of license
	license_file: string,	// maybe this is like icon, lets us know it exists and can use the link to the file.
	funding: string,	// URL for now, but later maybe array of objects? Or...?

	size: number	// bytes of what? compressed package? 
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
	migrations: MigrationStep[] = [];
    //schemas: number[] = [];

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
			const new_app_data = <AppManifest>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			Object.assign(this, new_app_data);
			//if( !this.schemas ) this.schemas = [];
		}
		catch(e) {
			m.sendError("error processing app version data "+e);
			console.error(e);
			return;
		}

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

	// get possible_migrations() {
	// 	if( this.schemas.length === 0 ) return [];
	// 	const lowest = this.schemas[0];
	// 	return [lowest-1, ...this.schemas];
	// }
}

const appData = reactive(new AppData());
appData._start();
export default appData;
