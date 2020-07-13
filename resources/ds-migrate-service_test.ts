import * as path from "https://deno.land/std/path/mod.ts";
import { assertEquals } from "https://deno.land/std/testing/asserts.ts";
import { stub, Stub } from "https://raw.githubusercontent.com/udibo/mock/v0.3.0/stub.ts";
import Twine, {Message} from './twine/twine.ts';
import Metadata from './ds-metadata.ts';
import {runMigration, runStep} from './ds-migrate-service.ts';

// test payload read
// 

Deno.test({
	name: "run step error",
	fn: async () => {
		
		const dir = await Deno.makeTempDir();
		const mig_dir = path.join(dir, "migrations", "2");
		await Deno.mkdir(mig_dir, {recursive: true});

		const ts = 'export default function migrateUp() { zzzl; };';
		await Deno.writeFile(path.join(mig_dir, "up.ts"), new TextEncoder().encode(ts));

		Metadata.app_path = dir;

		let err:Error|undefined = undefined;
		
		try {
			await runStep(2, true);
		}
		catch(e) {
			err = e;
		}
		
		if(!err) {
			throw new Error("no error thrown for bad test");
		}

		await Deno.remove(dir, {recursive: true});
	}
});

Deno.test({
	name: "run step up",
	fn: async () => {
		const dir = await Deno.makeTempDir();
		const mig_dir = path.join(dir, "migrations", "2");
		await Deno.mkdir(mig_dir, {recursive: true});

		const ts = 'export default function migrateUp() {};';
		await Deno.writeFile(path.join(mig_dir, "up.ts"), new TextEncoder().encode(ts));

		Metadata.app_path = dir;

		await runStep(2, true);

		await Deno.remove(dir, {recursive: true});
	}
});

Deno.test("run migration up", async ()=> {
	const dir = await Deno.makeTempDir();
	const mig_dir = path.join(dir, "migrations");
	await Deno.mkdir(path.join(mig_dir, "2"), {recursive: true});
	await Deno.mkdir(path.join(mig_dir, "3"));

	for( const n of [2,3] ) {
		const ts = 'export default async function migrateUp() { await Deno.writeFile("'+dir+'/out.txt", new TextEncoder().encode("'+n+'"), {append: true}); }';	// no-op migrate script. Could improve by having the migration code leave a trace of its run, like a file.
		await Deno.writeFile(path.join(mig_dir, n+"", "up.ts"), new TextEncoder().encode(ts));
	}

	Metadata.app_path = dir;

	const payload_data = {from: 1, to:3};

	const twine = new Twine("", false);
	const msg = new Message({
		service: 13,
		command: 11,
		msgID: 99,
		refMsgID: 0,
		payload: new TextEncoder().encode(JSON.stringify(payload_data))
	}, twine);

	const stubbed_sendOK: Stub<Message> = stub(msg, "sendOK");
	const stubbed_sendError: Stub<Message> = stub(msg, "sendError");

	await runMigration(msg);

	if( stubbed_sendError.calls.length > 0 ) {
		console.error(stubbed_sendError.calls[0].args[0])
	}

	assertEquals(stubbed_sendOK.calls.length, 1);

	const f_data = await Deno.readFile(path.join(dir, 'out.txt'))
	assertEquals(new TextDecoder().decode(f_data), "23");
	
	await Deno.remove(dir, {recursive: true});
});


Deno.test("run migration down", async ()=> {
	const dir = await Deno.makeTempDir();
	const mig_dir = path.join(dir, "migrations");
	await Deno.mkdir(path.join(mig_dir, "2"), {recursive: true});
	await Deno.mkdir(path.join(mig_dir, "3"));

	for( const n of [2,3] ) {
		const ts = 'export default async function migrateUp() { await Deno.writeFile("'+dir+'/out.txt", new TextEncoder().encode("'+n+'"), {append: true}); }';	// no-op migrate script. Could improve by having the migration code leave a trace of its run, like a file.
		await Deno.writeFile(path.join(mig_dir, n+"", "down.ts"), new TextEncoder().encode(ts));
	};

	Metadata.app_path = dir;

	const payload_data = {from:3, to:1};

	const twine = new Twine("", false);
	const msg = new Message({
		service: 13,
		command: 11,
		msgID: 99,
		refMsgID: 0,
		payload: new TextEncoder().encode(JSON.stringify(payload_data))
	}, twine);

	const stubbed_sendOK: Stub<Message> = stub(msg, "sendOK");
	const stubbed_sendError: Stub<Message> = stub(msg, "sendError");

	await runMigration(msg);

	if( stubbed_sendError.calls.length > 0 ) {
		console.error(stubbed_sendError.calls[0].args[0])
	}

	assertEquals(stubbed_sendOK.calls.length, 1);

	const f_data = await Deno.readFile(path.join(dir, 'out.txt'))
	assertEquals(new TextDecoder().decode(f_data), "32");
	
	await Deno.remove(dir, {recursive: true});
});