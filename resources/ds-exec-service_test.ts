import { assertEquals } from "https://deno.land/std/testing/asserts.ts";
import { stub, Stub } from "https://raw.githubusercontent.com/udibo/mock/v0.3.0/stub.ts";
import * as path from "https://deno.land/std/path/mod.ts";
import Twine, { Message } from "./twine/twine.ts";
import {handleMessage} from "./ds-exec-service.ts";

Deno.test({
	name: "execute default function",
	//ignore: true,
	fn: async () => {

		const file = "testFile.ts";
		const dir = await Deno.makeTempDir();
		const code = "export default function testFn() {};";
		
		const full_path = path.join(dir, file);
		await Deno.writeFile(full_path, new TextEncoder().encode(code));

		const twine = new Twine("", false);

		const send_data = {	file: full_path	};

		const m = new Message({
			service: 12,
			command: 11,
			msgID: 133,
			refMsgID: 0,
			payload:  new TextEncoder().encode(JSON.stringify(send_data))
		}, twine);

		const stubbed_sendOK: Stub<Message> = stub(m, "sendOK");

		await handleMessage(m);

		assertEquals(stubbed_sendOK.calls.length, 1);
		stubbed_sendOK.restore();
	}
});

Deno.test({
	name: "execute named function",
	//ignore: true,
	fn: async () => {

		const file = "testFile.ts";
		const dir = await Deno.makeTempDir();
		const code = "export function testFn() {};";
		
		const full_path = path.join(dir, file);
		await Deno.writeFile(full_path, new TextEncoder().encode(code));

		const twine = new Twine("", false);

		const send_data = {	file: full_path, fn: "testFn" };

		const m = new Message({
			service: 12,
			command: 11,
			msgID: 133,
			refMsgID: 0,
			payload:  new TextEncoder().encode(JSON.stringify(send_data))
		}, twine);

		const stubbed_sendOK: Stub<Message> = stub(m, "sendOK");

		await handleMessage(m);

		assertEquals(stubbed_sendOK.calls.length, 1);
		stubbed_sendOK.restore();
	}
});