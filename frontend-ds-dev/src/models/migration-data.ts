import {reactive} from 'vue';
import twineClient from './twine-client';
import {ReceivedMessageI} from '../twine-ws/twine-common';

const route_commands = {
	migration_event: 11,
};

type MigrationStatusData = {
	job_id: number
	appspace_id: number
	status: number
	started: Date|null
	finished: Date|null
	err: string|null
	cur_schema: number
}

// MigrationData records migration jobs and their updates.
class MigrationData {
	jobs :MigrationStatusData[];
	constructor() {
		twineClient.registerService(14, this);
		this.jobs = reactive([]);
	}
	handleMessage(m:ReceivedMessageI) {
		switch(m.command){
			case route_commands.migration_event:
				this.handleMigrationEvent(m);
			break;
			default:
				m.sendError("unrecognized service");
		}
	}
	handleMigrationEvent(m:ReceivedMessageI) {
		try {
			const event = <MigrationStatusData>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			const job_id = event.job_id;
			const index = this.jobs.findIndex((j:MigrationStatusData) => j.job_id === job_id );
			if( index === -1 ) this.jobs.push(event);
			else this.jobs[index] = event;
		}
		catch(e) {
			m.sendError("error processing migration event "+e);
			console.error(e);
			return;
		}

		m.sendOK();

		console.log("jobs", this.jobs);
	}
}

const migrationData = new MigrationData();

export default migrationData;
