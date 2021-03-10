import TwineWebsocketClient from '@/vendor/twine-ws';
import twineClient from '../twine-services/twine_client';
import {ReceivedMessageI, SentMessageI} from '../vendor/twine-ws/twine-common';

import {get, post} from '../controllers/userapi';

// MigrationJobResp describes a pending or ongoing appspace migration job
// type MigrationJobResp struct {
// 	JobID      domain.JobID         `json:"job_id"`
// 	OwnerID    domain.UserID        `json:"owner_id"`
// 	AppspaceID domain.AppspaceID    `json:"appspace_id"`
// 	ToVersion  domain.Version       `json:"to_version"`
// 	Created    time.Time            `json:"created"`
// 	Started    nulltypes.NullTime   `json:"started"`
// 	Finished   nulltypes.NullTime   `json:"finished"`
// 	Priority   bool                 `json:"priority"`
// 	Error      nulltypes.NullString `json:"error"`
// }

const remoteService = 12;

const remoteSubscribe = 11;
const remoteAppspaceSubscribe = 12;
const remoteUnsubscribe = 13;

export class MigrationJob {
	loaded = false;

	job_id = 0; 
	owner_id = 0;
	appspace_id = 0;
	to_version = "";
	created = new Date;
	started : null | Date = null;
	finished : null | Date = null;
	priority = false;
	error : string | null = null;

	setFromRaw(raw :any) {
		this.job_id = Number(raw.job_id);
		this.owner_id = Number(raw.owner_id);
		this.appspace_id = Number(raw.appsapce_id);
		this.to_version = raw.to_version + "";
		this.created = new Date(raw.created);
		this.started = raw.started ? new Date(raw.started) : null;
		this.finished = raw.finished ? new Date(raw.finished) : null;
		this.priority = !!raw.priority;
		this.error = raw.error ? raw.error+"" : null;

		this.loaded = true;
	}
}

export async function createMigrationJob(appspace_id:number, to_version:string) :Promise<MigrationJob> {
	const data = await post('/migration-job', {appspace_id, to_version});
	const job = new MigrationJob;
	job.setFromRaw(data);
	return job;
}

// For now MigrationJobs fetches and subscribes to changes for a given appspace.
// Later we can expand on this.
export class MigrationJobs {
	appspace_id : number|undefined;

	jobs :Map<number, MigrationJob> = new Map();

	subMessage :SentMessageI|undefined;

	async fetchForAppspace(appspace_id:number) {
		this.appspace_id = appspace_id;
		const resp_data = await get('/migration-job?appspace_id='+appspace_id);
		resp_data.forEach( (raw:any) => {
			const job = new MigrationJob;
			job.setFromRaw(raw);
			this.jobs.set(job.job_id, job);
		});
	}

	async connect(appspace_id:number) {
		if( this.appspace_id !== undefined && this.appspace_id !== appspace_id ) return;
		this.appspace_id = appspace_id;

		await this.disconnect();

		const payload = new TextEncoder().encode(JSON.stringify({appspace_id: this.appspace_id}));

		await twineClient.ready();
		this.subMessage = await twineClient.twine.send(remoteService, remoteAppspaceSubscribe, payload);

		for await (const m of this.subMessage.incomingMessages()) {
			switch (m.command) {
				case 11:
					const raw :any = JSON.parse(new TextDecoder('utf-8').decode(m.payload));
					const job_id = Number(raw.job_id);
					if( !this.jobs.has(job_id) ) this.jobs.set(job_id, new MigrationJob);
					const job = this.jobs.get(job_id);
					job!.setFromRaw(raw);
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
		await this.subMessage.refSendBlock(remoteUnsubscribe, undefined);
		this.subMessage = undefined;
		this.appspace_id = undefined;
	}
}