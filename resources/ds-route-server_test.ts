import * as path from "https://deno.land/std@0.97.0/path/mod.ts";
import { sleep } from "https://deno.land/x/sleep/mod.ts";
import { assertEquals } from "https://deno.land/std@0.97.0/testing/asserts.ts";
import { stub, Stub } from "https://raw.githubusercontent.com/udibo/mock/v0.8.0/stub.ts";
import Twine from "./twine/twine.ts";
import DsServices from "./ds-services.ts";
import Metadata from "./ds-metadata.ts";

// NOTE: these tests fail and will fail until deno fixes:
// https://github.com/denoland/deno/issues/8272

Deno.test({
	name: "start and stop server",
	ignore: true,	// Probably need to put more thought into how the app router is loaded to make this test worthwhile.
	fn: async () => {
		const t = new Twine("", false);
		//@ts-ignore
		DsServices.twine = t;
		const stubbed_sendBlock: Stub<Twine> = stub(t, "sendBlock");
		stubbed_sendBlock.returns = [{ok:true}];

		const dir = await Deno.makeTempDir();

		const orig_sock_path = Metadata.sock_path;
		Metadata.sock_path = dir;

		const orig_app_path = Metadata.app_path;
		Metadata.app_path = "/"; //???

		const serv_mod = await import('./ds-route-server.ts');
		const s = serv_mod.default;
		
		await s.startServer();

		const calls = stubbed_sendBlock.calls;
		assertEquals(calls.length, 1);

		await sleep(1);
		// would love to send a request just to prove it works
		// but wouldn't it try to get an app code, etc...? seems heavy.

		await s.stopServer();

		stubbed_sendBlock.restore();
		Metadata.sock_path = orig_sock_path;
	}
});

Deno.test({
	name: "start and stop server with twine",
	ignore: true,
	fn: async () => {
		const dir = await Deno.makeTempDir();

		const twine_sock = path.join(dir, "rev.sock");
		const host_twine = new Twine(twine_sock, true);
		const host_twine_start = host_twine.startServer();
		const sandbox_twine = new Twine(twine_sock, false);
		await sandbox_twine.startClient();
		//@ts-ignore
		DsServices.twine = sandbox_twine;

		await host_twine_start;

		(async function() {
			for await( let m of host_twine.incomingMessages() ) {
				if( m.service == 11 && m.command == 11 ) {
					console.log("got server ready");
					m.sendOK();
				}
				else {
					console.error("What essage is this?");
					throw new Error("What is this message? "+m.service+' '+m.command);
				}
			}
		})();

		const orig_sock_path = Metadata.sock_path;
		Metadata.sock_path = dir;

		const serv_mod = await import('./ds-route-server.ts');
		const s = serv_mod.default;

		(async function() {
			for await( let m of sandbox_twine.incomingMessages() ) {
				if( m.service == 11 && m.command == 13 ) {
					console.log("got message to stop server");
					try {
						// All we need to do is stop the route server, and the script will exit. I think.
						await s.stopServer();
					}
					catch(e) {
						m.sendError(e);
					}
					m.sendOK();
				}
			}
		})();
		
		await s.startServer();

		// would love to send a request just to prove it works
		// but wouldn't it try to get an app code, etc...? seems heavy.

		await sleep(1);

		console.log("Going to stop server now");
		const reply = await host_twine.sendBlock(11, 13, undefined);
		if( reply.error ) throw reply.error;

		Metadata.sock_path = orig_sock_path;
	}
});