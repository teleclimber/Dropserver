import * as path from "https://deno.land/std@0.106.0/path/mod.ts";

import type {ReceivedMessageI} from "./twine.ts";

import Metadata from './ds-metadata.ts';

const run_migration_cmd = 11;

type MigrationData = {
	from: number,
	to: number
};

export async function handleMessage(message :ReceivedMessageI) {
	switch (message.command) {
		case run_migration_cmd:
			await runMigration(message);
			break;
	
		default:
			await message.sendError("Command not recognized");
	}
}

export async function runMigration(message :ReceivedMessageI) {
	let migration_data:MigrationData;
	try {
		migration_data = readMigrationPayload(message.payload)
	}
	catch(e) {
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

	if( from_schema < to_schema ) {
		for( let i=from_schema+1; i<=to_schema; ++i ) {
			//console.log( 'running up migration for '+i ); // maybe this is an update message to host
			try {
				await runStep(i, true);
			}
			catch(e) {
				await message.sendError("Error running migration step: "+i+', '+e.toString());
				return;
			}
		}
	}
	else {
		// contrary to up, going down means running down.js at current level, and stopping short of desired level
		for( let i=from_schema; i>to_schema; --i ) {
			//console.log( 'running down migration for '+i );
			try {
				await runStep(i, false);
			}
			catch(e) {
				await message.sendError("Error running migration step: "+i+', '+e.toString());
				return;
			}
		}
	}

	message.sendOK();
}

export async function runStep(num:number, up:boolean) {
	const mod_path = path.join( Metadata.app_path, "migrations", num+'', up?"up.ts":"down.ts" );

	let mod:any;
	try {
		mod = await import( mod_path );	// straight up node module for now
		// ^^ could also use import, and I think we will have to,
		// .. but not clear how we deal with async migration code (which it ofetn will be)
		// Also all this naming convention is a little offputting.
	}
	catch(e) {
		// failed to require the migration module
		throw new Error("failed to require module path: "+mod_path+', '+e);
	}
	
	try {
		await mod.default();
	}
	catch( e ) {
		throw new Error("failed to execute migration function, module path: "+mod_path+', '+e);
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