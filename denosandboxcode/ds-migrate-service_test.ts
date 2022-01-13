import * as path from "https://deno.land/std@0.106.0/path/mod.ts";
import { assertEquals } from "https://deno.land/std@0.106.0/testing/asserts.ts";
import { stub, Stub } from "https://raw.githubusercontent.com/udibo/mock/v0.8.0/stub.ts";
import Twine, {Message} from './twine.ts';
import Migrations from './migrations.ts';
import MigrationService from './ds-migrate-service.ts';

// test payload read

Deno.test({
	name: "run step with JS error in step",
	fn: async () => {
		const mFn = () => {
			//@ts-ignore because the point is to trigger an error
			return <Promise<void>>new Promise(zzl);
		}

		const migrations = new Migrations;
		migrations.setCallback( () => [{direction:"up", schema:2, func: mFn}] );
		await migrations.loadMigrations();

		const service = new MigrationService(migrations);

		let err:Error|undefined = undefined;
		
		try {
			await service.runStep(2, true);
		}
		catch(e) {
			err = e;
		}
		
		if(!err) {
			throw new Error("no error thrown for bad test");
		}
		else {
			console.log("OK got error:", err);
		}
	}
});

Deno.test({
	name: "run step up",
	fn: async () => {
		const mFn = () => {
			return <Promise<void>>new Promise( (resolve, _) => {resolve()});
		}

		const migrations = new Migrations;
		migrations.setCallback( () => [{direction:"up", schema:2, func: mFn}] );
		await migrations.loadMigrations();

		const service = new MigrationService(migrations);

		await service.runStep(2, true);
	}
});

// this is kind of an all-in test.
//.. need to inject app path in metadata somehow.
Deno.test("run migration up", async ()=> {
	const dir = await Deno.makeTempDir();

	const up2 = async () => {
		await Deno.writeFile(path.join(dir,'out.txt'), new TextEncoder().encode('2'), {append: true});
	}
	const up3 = async () => {
		await Deno.writeFile(path.join(dir,'out.txt'), new TextEncoder().encode('3'), {append: true});
	}

	const migrations = new Migrations;
	migrations.setCallback( () => [
		{direction:"up", schema:2, func: up2},
		{direction:"up", schema:3, func: up3}
	]);
	const service = new MigrationService(migrations);

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

	await service.runMigration(msg);

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

	const down2 = async () => {
		await Deno.writeFile(path.join(dir,'out.txt'), new TextEncoder().encode('2'), {append: true});
	}
	const down3 = async () => {
		await Deno.writeFile(path.join(dir,'out.txt'), new TextEncoder().encode('3'), {append: true});
	}

	const migrations = new Migrations;
	migrations.setCallback( () => [
		{direction:"down", schema:2, func: down2},
		{direction:"down", schema:3, func: down3}
	]);
	const service = new MigrationService(migrations);

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

	await service.runMigration(msg);

	if( stubbed_sendError.calls.length > 0 ) {
		console.error(stubbed_sendError.calls[0].args[0])
	}

	assertEquals(stubbed_sendOK.calls.length, 1);

	const f_data = await Deno.readFile(path.join(dir, 'out.txt'))
	assertEquals(new TextDecoder().decode(f_data), "32");
	
	await Deno.remove(dir, {recursive: true});
});