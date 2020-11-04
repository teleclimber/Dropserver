import DsServices from "./ds-services.ts";
import type Twine from "./twine/twine.ts";

// remote service ID
const service = 15;

// remote commands:
const createCmd = 11;
const deleteCmd = 13;
const queryCmd = 20;

export type ExecResults = {
	last_insert_id: number,
	rows_affected: number
}

export type QueryResults = {
	rows: any[]
}

type QueryData = {
	db_name: string,
	type: string,	// "query" or "exec"
	sql: string,
	params?: any[],
	named_params?: object
}


export async function createDatabase(db_name: string) :Promise<Database> {
	const twine = DsServices.getTwine();

	const reply = await twine.sendBlock(service, createCmd, makePayload({db_name}));
	if(reply.error) {
		throw new Error(reply.error);
	}

	return new Database(db_name);
}

export async function deleteDatabase(db_name: string) {
	const twine = DsServices.getTwine();

	const reply = await twine.sendBlock(service, deleteCmd, makePayload({db_name}));
	if(reply.error) {
		throw new Error(reply.error);
	}
}

export default class Database { 
	constructor(private db_name: string) {}

	async exec(sql: string, parameters?:any[]|object): Promise<ExecResults> {
		const twine = DsServices.getTwine();

		const q_data:QueryData = {
			db_name: this.db_name,
			type: "exec",
			sql: sql
		};

		if( Array.isArray(parameters) ) {
			q_data.params = parameters;
		} else if( typeof parameters === 'object' ) {
			q_data.named_params = parameters;
		}

		const reply = await twine.sendBlock(service, queryCmd, makePayload(q_data));
		if(reply.error) {
			throw new Error(reply.error);
		}

		const results = <ExecResults>decodePayload(reply.payload);

		reply.sendOK();

		return results;
	}

	async query(sql: string, parameters?:any[]|object): Promise<QueryResults> {
		const twine = DsServices.getTwine();

		const q_data:QueryData = {
			db_name: this.db_name,
			type: "query",
			sql: sql
		};

		if( Array.isArray(parameters) ) {
			q_data.params = parameters;
		} else if( typeof parameters === 'object' ) {
			q_data.named_params = parameters;
		}

		const reply = await twine.sendBlock(service, queryCmd, makePayload(q_data));
		if(reply.error) {
			throw new Error(reply.error);
		}

		const results = <QueryResults>decodePayload(reply.payload);

		reply.sendOK();

		return results;
	}
}

function makePayload(data:object):Uint8Array|undefined {
	if(data === undefined) return undefined;
	return new TextEncoder().encode(JSON.stringify(data));
}

function decodePayload(payload:Uint8Array|undefined) :undefined|object {
	return JSON.parse(new TextDecoder().decode(payload))
}

