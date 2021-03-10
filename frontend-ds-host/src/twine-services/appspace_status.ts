import twineClient from './twine_client';
import {ReceivedMessageI, SentMessageI} from '../vendor/twine-ws/twine-common';
import {reactive} from 'vue';

// type AppspaceStatusEvent struct {
//     AppspaceID       AppspaceID `json:"appspace_id"`
//     Paused           bool       `json:"paused"`
//     TempPaused       bool       `json:"temp_paused"`
//     Migrating        bool       `json:"migrating"`
//     AppspaceSchema   int        `json:"appspace_schema"`
//     AppVersionSchema int        `json:"app_version_schema"`
//     Problem          bool       `json:"problem"` // string? To hint at the problem?
// }

// export type AppspaceStatus = {
// 	loaded: boolean
// 	appspace_id: Number
// 	paused: boolean
// 	temp_paused: boolean
// 	migrating: boolean
// 	appspace_schema: Number
// 	app_version_schema: Number
// 	problem: boolean
// }

const remoteService = 11;

const remoteSubscribe = 11;
const remoteUnsubscribe = 13;

export class AppspaceStatus {
	loaded = false;
	appspace_id = -1;
	paused = false;
	temp_paused = false;
	migrating = false;
	appspace_schema = 0;
	app_version_schema = 0;
	problem = false;

	subMessage :SentMessageI|undefined;
	
	constructor() {}

	async connectStatus(appspace_id: number) {
		await this.disconnect();

		this.appspace_id = appspace_id;

		const payload = new TextEncoder().encode(JSON.stringify({appspace_id: this.appspace_id}));

		await twineClient.ready();
		this.subMessage = await twineClient.twine.send(remoteService, remoteSubscribe, payload);

		for await (const m of this.subMessage.incomingMessages()) {
			switch (m.command) {
				case 11:	//status update
					const raw :any = JSON.parse(new TextDecoder('utf-8').decode(m.payload));
					if( raw.appspace_id != this.appspace_id ) break;
					this.updateFromRaw(raw);
					this.loaded = true;
					console.log("status updated", this.appspace_id);
					m.sendOK();
					break;
			
				default:
					m.sendError("What is this command?");
					throw new Error("what is this command? "+m.command);
			}
		}
	}
	async disconnect() {
		if( this.subMessage === undefined ) return;
		await this.subMessage.refSendBlock(13, undefined);
		this.subMessage = undefined;
		this.appspace_id = -1;
	}
	updateFromRaw(raw:any) {
		this.appspace_id = Number(raw.appspace_id),
		this.paused = !!raw.paused,
		this.temp_paused = !!raw.temp_paused,
		this.migrating = !!raw.migrating,
		this.appspace_schema = Number(raw.appspace_schema),
		this.app_version_schema = Number(raw.app_version_schema),
		this.problem = !!raw.problem
	}
	get as_string() :string {
		if( !this.loaded ) return "loading";
		if( this.problem ) return "problem";
		if( this.migrating ) return "migrating";
		if( this.app_version_schema !== this.appspace_schema ) return "migrate";
		if( this.paused ) return "paused";
		if( this.temp_paused ) return "busy";
		return "ready";
	};
}
