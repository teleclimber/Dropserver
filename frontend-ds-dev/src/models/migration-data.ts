import {reactive} from 'vue';
import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

const route_commands = {
	migration_event: 11,
};

type MigrationStatusData = {
	job_id: number	// meh
	appspace_id: number	// irrelevant here
	started: Date|null	// who cares i nthis context
	finished: Date|null
	error: string|null
}

// MigrationData records migration jobs and their updates.
class MigrationData {
	last_job :MigrationStatusData|undefined;
	_start() {
		twineClient.registerService(14, this);	// may be unnecessary
		
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
			this.last_job = <MigrationStatusData>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
		}
		catch(e) {
			m.sendError("error processing migration event "+e);
			console.error(e);
			return;
		}

		m.sendOK();
	}
}

const migrationData = reactive(new MigrationData());
migrationData._start();

export default migrationData;
