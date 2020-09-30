import type { ReceivedMessageI } from "./twine/twine.ts";

const exec_fn_cmd = 11;

type ExecFnData = {
	file: string,
	fn?: string
}

export async function handleMessage(message: ReceivedMessageI) {
	switch (message.command) {
		case exec_fn_cmd:
			await execFnHandler(message);
			break;
	
		default:
			await message.sendError("Command not recognized");
	}
}

async function execFnHandler(message: ReceivedMessageI) {
	const payload_data = payloadToObj(message.payload);
	if( payload_data === undefined ) {
		await message.sendError("payload was undefined");
		return;
	}

	let mod:any;
	let data : ExecFnData;
	try {
		data = readExecFnData(payload_data);
		// load module. is path absolute?
		mod = await import(data.file);
	}
	catch(e) {
		await message.sendError(e.toString());
		return;
	}
	// we need more ways of saying what we want to execute
	// ..import a function? or a an obect with a method? What?

	let fn : () => void;
	if( data.fn === undefined ) fn = mod.default;
	else fn = mod[data.fn];

	if(typeof fn !== "function") {
		await message.sendError("Not a function: "+data.fn);
		return;
	}

	try {
		await fn();
		// you *could* pass in a form of message that allows ref sends and receives maybe ?
	}
	catch(e) {
		console.error(e);
		await message.sendError("Error caught while running script: "+e);	// Here error is specifically with script that was asked to run
		return;
	}

	await message.sendOK();
}

function readExecFnData(data:object) :ExecFnData {
	const ret = <ExecFnData>data;
	if( ret.file === undefined || !ret.file ) throw new Error("file missing from exec data");
	// fn is optional
	return ret;
}

function payloadToObj(payload: Uint8Array|undefined) :object|undefined {
	if( payload === undefined) return undefined;
	return JSON.parse(new TextDecoder().decode(payload));
}
