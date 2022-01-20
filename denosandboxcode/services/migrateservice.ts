import type {ReceivedMessageI} from "./twine.ts";

import Migrations from '../migrations.ts';

const run_migration_cmd = 11;
const get_migrations_cmd = 12;

type MigrationData = {
	from: number,
	to: number
};

export default class MigrationService {
	constructor(private migrations:Migrations) {}

	handleMessage(message :ReceivedMessageI) {
		switch (message.command) {
			case run_migration_cmd:
				this.runMigration(message);
				break;
			case get_migrations_cmd:
				this.getMigrations(message);
				break;
		
			default:
				message.sendError("Command not recognized");
		}
	}
	async getMigrations(message: ReceivedMessageI) {
		try {
			await this.migrations.loadMigrations();
		}
		catch(e) {
			console.error(e);
			await message.sendError("Error loading migrations: "+e);
			return;
		}

		const migrations = this.migrations.getMigrations();
		message.reply(11, new TextEncoder().encode(JSON.stringify(migrations)));
	}
	async runMigration(message :ReceivedMessageI) {
		let migration_data:MigrationData;
		try {
			migration_data = readMigrationPayload(message.payload)
		}
		catch(e) {
			console.error(e);
			await message.sendError("Error in payload: "+e);
			return;
		}
		
		const from_schema = migration_data.from;
		const to_schema = migration_data.to;
	
		if( from_schema === to_schema ) {
			// this shouldn't happen. Host should prevent
			await message.sendError("from and to schema numbers are the same");
			return;
		}

		// Here we actually want to load teh migrations
		// should wrap in a try-catch
		// Because this could error on user code.
		try {
			await this.migrations.loadMigrations();
		}
		catch(e) {
			console.error(e);
			await message.sendError("Error loading migrations: "+e);
			return;
		}
	
		if( from_schema < to_schema ) {
			for( let i=from_schema+1; i<=to_schema; ++i ) {
				try {
					await this.runStep(i, true);
				}
				catch(e) {
					console.error(e);
					await message.sendError("Error running migration step: "+i+', '+e.toString());
					return;
				}
			}
		}
		else {
			// contrary to up, going down means running down.js at current level, and stopping short of desired level
			for( let i=from_schema; i>to_schema; --i ) {
				try {
					await this.runStep(i, false);
				}
				catch(e) {
					console.error(e);
					await message.sendError("Error running migration step: "+i+', '+e.toString());
					return;
				}
			}
		}
	
		message.sendOK();
	}

	async runStep(num:number, up:boolean) {
		const func = this.migrations.getFunc(up, num);
		try {
			await func();
		}
		catch( e ) {
			throw new Error("failed to execute migration function: "+e);
		}
	}
}


function readMigrationPayload(payload: Uint8Array|undefined) :MigrationData {
	let data:any;
	try {
		data = payloadToObj(payload);
	}
	catch(e) {
		throw new Error("Error parsing json payload: "+e);
	}
	if( payload === undefined ) {
		throw new Error("payload was undefined");
	}

	if( data.from !== 0 && !data.from ) {
		throw new Error("Field from not provided");
	}
	if( data.to !== 0 && !data.to ) {
		throw new Error("Field to not provided");
	}
	
	return  {
		from: Number(data.from),
		to: Number(data.to)
	};
}
function payloadToObj(payload: Uint8Array|undefined) :object|undefined {
	if( payload === undefined) return undefined;
	return JSON.parse(new TextDecoder().decode(payload));
}