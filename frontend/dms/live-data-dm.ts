// somewhat temporary all-encompassing live-data DM.
// Will probably have to break it up into
// - websocket helper lib
// - a connection to live-data on ds-host
// - different live-dms for different types of live data (like jobs versus appspace sandbox status?) 

import ds_axios, {url} from '../ds-axios-helper-ts';

import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import {GetStartLiveDataResp} from '../generated-types/userroutes-classes';

export enum MigrationStatus { "started", "running", "finished" }
function hydrateMigrationStatus(s:string) {
	switch(s) {
		case "started": return MigrationStatus.started;
		case "running": return MigrationStatus.running;
		case "finished": return MigrationStatus.finished;
		default: throw new Error("what is this migration status? "+s);
	}
}

// more hydration fns
function hydrateNullDate(d:string|null) {
	if( typeof d === 'string' ) {
		return new Date(d);
	}
	else return null;
}

export type LiveMigrationJob = {
	_is_dummy: boolean
	job_id: string
	
	owner_id: number
	appspace_id: number
	to_version: string
	created: Date
	priority: boolean

	status: MigrationStatus
	started: Date | null
	finished: Date | null
	error: string | null
	cur_schema: number
}
function makeLiveJobDummy() {
	return {
		_is_dummy: true,
		job_id: "",
	
		owner_id: 0,
		appspace_id: 0,
		to_version: "",
		created: new Date,
		priority: false,

		status: MigrationStatus.started,
		started: null,
		finished: null,
		error: null,
		cur_schema: 0
	}
}

export default class LiveDataDM {
	@observable jobs: {[job_id:string]:LiveMigrationJob} = {};
	@observable connected = false;

	constructor() {}

	async connect() {
		// send a request for token via axios via axios 
		// then connect to ws
		
		let resp :any;
		try {
			resp = await ds_axios.get( '/live' );
		}
		catch(error) {
			if( error.response.status == 401 ) window.location.href = '/login';
			else throw new Error( error );
			return;
		}

		if( !resp || !resp.data ) return;

		const tok_resp = new GetStartLiveDataResp(resp.data);

		if( !url.startsWith("//") ) {
			throw new Error("unexpected form of user routes base url");
		}

		let wss_protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';

		let socket = new WebSocket(wss_protocol+url+"/live/"+tok_resp.token)

		socket.onopen = () => {
			runInAction( () => this.connected = true )
		};
		  
		socket.onmessage = (event) => {
			if( !event.data ) return;
			this.hydrateJob(JSON.parse(event.data));
		};
		  
		socket.onclose = (event) => {
			runInAction( () => this.connected = false )
			if (event.wasClean) {
				console.log("[close] Connection closed cleanly", event);
			} else {
				// e.g. server process killed or network down
				// event.code is usually 1006 in this case
				console.log('[close] Connection died');
			}

			// TODO: reconnect?
		};
		  
		socket.onerror = function(error) {
			console.error(error);
		};

		window.addEventListener('beforeunload', function(){
			socket.close();
		});
	}

	@action
	hydrateJob(job_data:any) {
		const job_id = job_data.job_id+'';

		if( !this.jobs[job_id] ) {
			this.jobs[job_id] = makeLiveJobDummy();
		}
		
		const lj = this.jobs[job_id];

		if( lj._is_dummy ) {
			if( !job_data.migration_job ) {
				throw new Error("got an update for a job we do not have");
			}
			const mj = job_data.migration_job;
			lj._is_dummy = false;
			lj.job_id = job_id;
			lj.owner_id = mj.owner_id;
			lj.appspace_id = mj.appspace_id;
			lj.created = new Date(mj.created);
			lj.to_version = mj.to_version;
			lj.priority = mj.priority;
		}
		lj.status = hydrateMigrationStatus(job_data.status);
		lj.started = hydrateNullDate(job_data.started);
		lj.finished = hydrateNullDate(job_data.finished);
		lj.error = job_data.error;
		lj.cur_schema = job_data.cur_schema
	}

	@action
	getJob(job_id:string) :LiveMigrationJob {
		// if it doesn't exist create a dummy, add it to list and return it.
		// then update it as data comes in.
		if( !this.jobs[job_id] ) {
			this.jobs[job_id] = makeLiveJobDummy();
		}
		return this.jobs[job_id];
	}

	// I think maybe running is useful, but I could be wrong.
	@computed get running() {
		const ret:{[appspace_id:string]:LiveMigrationJob} = {};
		for( let job_id in this.jobs ) {
			const j = this.jobs[job_id];
			if( j.status === MigrationStatus.started || j.status === MigrationStatus.running ) {
				ret[j.appspace_id] = j;	// implies that two running jobs on same appspace id is not expected.
			}
		}
		return ret;
	}

	@computed get appspaceJobs() {
		const ret:{[appspace_id:string]:LiveMigrationJob[]} = {};
		for( let job_id in this.jobs ) {
			const j = this.jobs[job_id];
			const appspace_id = j.appspace_id
			if( typeof ret[appspace_id] == undefined ) ret[appspace_id] = [];
			ret[appspace_id].push(j);
		}
		return ret;
	}
	
	// getActiveForAppspace(appspace_id: number) {	// wait what if there are several? Like one that finished and one that is running?
	// 	const ret :LiveMigrationJob[] = [];
	// 	for( let job_id in this.live_jobs) {
	// 		const j = this.live_jobs[job_id];
	// 		if( !j._is_dummy && j.appspace_id === appspace_id && j.status !== MigrationStatus.finished ) {
	// 			ret.push(j);
	// 		}
	// 	}
	// 	return ret;
	// 	// Really not clear how this is observable?!?!?!
	// }

}