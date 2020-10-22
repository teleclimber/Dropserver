import Twine from "./twine.ts";
import * as path from "https://deno.land/std/path/mod.ts";
import { assertEquals, assert } from "https://deno.land/std/testing/asserts.ts";

Deno.test({
	name: "send",
	//ignore: true,
	fn: async () => {
		const temp_dir = await Deno.makeTempDir();
		const sock_path = path.join(temp_dir, "test.sock");
		const t = new Twine(sock_path, false);

		// warning unstable API in Deno. Use "--unstable"
		const s_listener = await Deno.listen({path: sock_path, transport:"unix"});

		const 
			service = 7,
			command = 11,
			msgID = 13,
			payload_str = "hello world";

		(async function() {
			const s_conn = await s_listener.accept();
			const buf = new Uint8Array(100);
			const numBytes = await s_conn.read(buf)
			const view = new DataView(buf.buffer);

			assertEquals(service, view.getInt8(0));
			assertEquals(command, view.getInt8(1));
			assertEquals(msgID, view.getInt8(2));

			const p_size = view.getInt16(3);
			assertEquals(11, p_size);

			const rcv_payload_str = new TextDecoder().decode(buf.slice(5, 5+p_size));
			assertEquals(payload_str, rcv_payload_str);

			s_conn.close();
			s_listener.close();
		})();

		// @ts-ignore setting private t.conn externally
		t.conn = await Deno.connect({ path: sock_path, transport: "unix" });

		// @ts-ignore _send is private
		await t._send(msgID, 0, service, command, new TextEncoder().encode(payload_str));


		t.close();
		

		await Deno.remove(temp_dir, {recursive: true});

	}
});

Deno.test({
	name: "receive",
	//ignore: true,
	fn: async () => {
		const payload_str = "hello world";

		const temp_dir = await Deno.makeTempDir();
		const sock_path = path.join(temp_dir, "test.sock");
		const twine_client = new Twine(sock_path, false);

		const twine_server = new Twine(sock_path, true);

		const s_listener = await Deno.listen({path: sock_path, transport:"unix"});
		let s_conn : Deno.Conn;

		(async function() {
			s_conn = await s_listener.accept();

			//@ts-ignore conn is private
			twine_server.conn = s_conn;

			//@ts-ignore _send is private
			await twine_server._send(128, 0, 13, 11, new TextEncoder().encode(payload_str) );

			await s_conn.close();
			await s_listener.close();
		})();

		// @ts-ignore setting private t.conn externally
		twine_client.conn = await Deno.connect({ path: sock_path, transport: "unix" });
		twine_client.receive();

		for await (const message of twine_client.incomingMessages() ) {
			assertEquals(message.msgID, 128);
			assertEquals(message.service, 13);
			assertEquals(message.command, 11);

			const rcv_payload_str = new TextDecoder().decode(message.payload);
			assertEquals(payload_str, rcv_payload_str);

			console.log("got incoming message");

			await twine_client.close();
		}
		await Deno.remove(temp_dir, {recursive: true});

		console.log("reached end of test");
	}
});

// Now I want to test full cycle: hi, message, graceful.
Deno.test({
	name: "hi, message, graceful",
	//ignore: true,
	fn: async () => {
		const temp_dir = await Deno.makeTempDir();
		const sock_path = path.join(temp_dir, "test.sock");

		const twine_client = new Twine(sock_path, false);
		const twine_server = new Twine(sock_path, true);

		const server_start = twine_server.startServer();
		await twine_client.startClient();
		await server_start;

		(async function() {
			for await (const message of twine_server.incomingMessages()) {
				console.log("got message");
				assertEquals(7, message.service);
				message.sendOK();
			}
		})();

		console.log("after start client");

		const reply = await twine_client.sendBlock(7, 11, undefined);
		assert(reply.ok);

		console.log("got OK");

		await twine_client.graceful();

		await Deno.remove(temp_dir, {recursive: true});
	}
});

Deno.test({
	name: "hi, multiple messages, from server, graceful",
	//ignore: true,
	fn: async () => {
		const temp_dir = await Deno.makeTempDir();
		const sock_path = path.join(temp_dir, "test.sock");

		const twine_client = new Twine(sock_path, false);
		const twine_server = new Twine(sock_path, true);

		const server_start = twine_server.startServer();
		await twine_client.startClient();
		await server_start;

		(async function() {
			for await (const message of twine_client.incomingMessages()) {
				console.log("got message");
				assertEquals(7, message.service);
				message.sendOK();
			}
		})();

		console.log("after start client");

		const replyP1 = twine_server.sendBlock(7, 11, new TextEncoder().encode("test payload numero uno"));
		const replyP2 = twine_server.sendBlock(7, 12, new TextEncoder().encode("test payload numero dos"));

		let reply = await replyP1;
		assert(reply.ok);
		reply = await replyP2;
		assert(reply.ok);

		console.log("got OK");

		await twine_server.graceful();

		await Deno.remove(temp_dir, {recursive: true});
	}
});


Deno.test({
	name: "hi, messages with reply and ok, from server, graceful",
	//ignore: true,
	fn: async () => {
		const temp_dir = await Deno.makeTempDir();
		const sock_path = path.join(temp_dir, "test.sock");

		const twine_client = new Twine(sock_path, false);
		const twine_server = new Twine(sock_path, true);

		const server_start = twine_server.startServer();
		await twine_client.startClient();
		await server_start;

		(async function() {
			for await (const message of twine_client.incomingMessages()) {
				console.log("got message");
				assertEquals(7, message.service);
				message.reply(77, undefined);
			}
		})();

		console.log("after start client");

		const replyP1 = twine_server.sendBlock(7, 11, new TextEncoder().encode("test payload numero uno"));
		const replyP2 = twine_server.sendBlock(7, 12, new TextEncoder().encode("test payload numero dos"));

		let reply = await replyP1;
		//assert(reply.ok);
		assertEquals(reply.command, 77);
		reply.sendOK();

		reply = await replyP2;
		//assert(reply.ok);
		assertEquals(reply.command, 77);
		reply.sendOK();

		console.log("got OK");

		await twine_server.graceful();

		await Deno.remove(temp_dir, {recursive: true});
	}
});

Deno.test({
	name: "ref-requests",
	//ignore: true,
	fn: async () => {
		const temp_dir = await Deno.makeTempDir();
		const sock_path = path.join(temp_dir, "test.sock");

		const twine_client = new Twine(sock_path, false);
		const twine_server = new Twine(sock_path, true);

		const server_start = twine_server.startServer();
		await twine_client.startClient();
		await server_start;

		(async function() {
			let num = 0;
			for await (const message of twine_server.incomingMessages()) {
				++num;
				assertEquals(7, message.service);

				for await (const ref_m of message.incomingMessages() ) {
					assertEquals(33, ref_m.command);
					ref_m.sendOK();
					message.sendOK();
				}

				
			}
			if(num !== 1) throw new Error("expected only one message");
		})();

		console.log("after start client");

		const sent = await twine_client.send(7, 11, undefined);
		console.log("after sent");
		const ref_rep = await sent.refSendBlock(33, undefined);
		assert(ref_rep.ok);
		console.log("after ref send block");

		const reply = await sent.waitReply();
		assert(reply.ok);

		console.log("got OK");

		await twine_client.graceful();

		await Deno.remove(temp_dir, {recursive: true});
	}
});

Deno.test({
	name: "pre-conn message, graceful",
	//ignore: true,
	fn: async () => {
		const temp_dir = await Deno.makeTempDir();
		const sock_path = path.join(temp_dir, "test.sock");

		const twine_client = new Twine(sock_path, false);
		const twine_server = new Twine(sock_path, true);

		const sent = await twine_client.send(7, 11, undefined);
		//assert(reply.ok);

		const server_start = twine_server.startServer();
		await twine_client.startClient();
		await server_start;

		(async function() {
			for await (const message of twine_server.incomingMessages()) {
				console.log("got message");
				assertEquals(7, message.service);
				message.sendOK();
			}
		})();

		console.log("after start client");

		const reply = await sent.waitReply();
		assert(reply.ok);

		console.log("got OK");

		await twine_client.graceful();

		await Deno.remove(temp_dir, {recursive: true});
	}
});