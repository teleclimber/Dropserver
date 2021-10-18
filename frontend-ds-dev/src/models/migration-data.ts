import {reactive} from 'vue';
import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

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
		twineClient.registerService(14, this);	// may be unnecessary
		this.jobs = reactive([]);
		
		this.handleRefMessages();
	}
	async handleRefMessages() {
		await twineClient.ready();
		const payload = new TextEncoder().encode(JSON.stringify({appspace_id:15}))
		const sent = await twineClient.twine.send(14, 12, payload);
		for await (const m of sent.incomingMessages()) {
			switch (m.command) {
				case route_commands.migration_event:
					this.handleMigrationEvent(m);
					break;
			
				default:
					m.sendError("What is this command?");
					throw new Error("what is this command? "+m.command);
			}
		}
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
	}
}

const migrationData = new MigrationData();

export default migrationData;
