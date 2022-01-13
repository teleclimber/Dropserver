import * as path from "https://deno.land/std@0.106.0/path/mod.ts";
import { sleep } from "https://deno.land/x/sleep/mod.ts";
import { assertEquals } from "https://deno.land/std@0.106.0/testing/asserts.ts";
import { stub, Stub } from "https://raw.githubusercontent.com/udibo/mock/v0.8.0/stub.ts";
import Twine from "./twine.ts";
import AppRoutes from './app-router.ts';
import DsServices from "./ds-services.ts";
import DsRouteServer from './ds-route-server.ts';

Deno.test({
	name: "start and stop server",
	fn: async () => {
		const dir = await Deno.makeTempDir();

		const dsServices = new DsServices();
		
		const t = new Twine("", false);
		const stubbed_sendBlock: Stub<Twine> = stub(t, "sendBlock");
		stubbed_sendBlock.returns = [{ok:true}];
		//@ts-ignore because private
		dsServices.twine = t;

		const appRoutes = new AppRoutes;

		const routeServer = new DsRouteServer(dsServices, appRoutes);

		await routeServer.startServer(dir);

		const calls = stubbed_sendBlock.calls;
		assertEquals(calls.length, 1);

		await sleep(1);
		// would love to send a request just to prove it works
		// but wouldn't it try to get an app code, etc...? seems heavy.

		await routeServer.stopServer();

		stubbed_sendBlock.restore();
	}
});

Deno.test({
	name: "start and stop server with twine",
	//ignore: true,
	fn: async () => {
		const dir = await Deno.makeTempDir();

		const dsServices = new DsServices();

		const twine_sock = path.join(dir, "rev.sock");
		const host_twine = new Twine(twine_sock, true);
		const host_twine_start = host_twine.startServer();
		const sandbox_twine = new Twine(twine_sock, false);
		await sandbox_twine.startClient();
		//@ts-ignore because private prop access
		dsServices.twine = sandbox_twine;

		await host_twine_start;

		(async function() {
			for await( const m of host_twine.incomingMessages() ) {
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

		const appRoutes = new AppRoutes;

		const routeServer = new DsRouteServer(dsServices, appRoutes);

		(async function() {
			for await( const m of sandbox_twine.incomingMessages() ) {
				if( m.service == 11 && m.command == 13 ) {
					console.log("got message to stop server");
					try {
						// All we need to do is stop the route server, and the script will exit. I think.
						await routeServer.stopServer();
					}
					catch(e) {
						m.sendError(e);
					}
					m.sendOK();
				}
			}
		})();
		
		await routeServer.startServer(dir);

		// would love to send a request just to prove it works
		// but wouldn't it try to get an app code, etc...? seems heavy.

		await sleep(1);

		console.log("Going to stop server now");
		const reply = await host_twine.sendBlock(11, 13, undefined);
		if( reply.error ) throw reply.error;

		await host_twine.graceful();
	}
});