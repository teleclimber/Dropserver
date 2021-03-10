// Consider the possibility that host will push data to a service
// that is not instantiated yet?
// How would that work? Can it even work?

// Well it kind of has. Example: cron.
// - receive message saying run function x at file y

import Twine from "./twine/twine.ts";
import Metadata from "./ds-metadata.ts";
import DsRouteServer from "./ds-route-server.ts";

import type {ReceivedMessageI} from './twine/twine.ts';

const sandboxService = 11;
const executeService = 12;
const migrateService = 13;

export class DsServices {
	private twine:Twine|undefined;
	constructor() {}

	async initTwine() {
		if(this.twine !== undefined) throw new Error("Twine already initiated");
		this.twine = new Twine(Metadata.rev_sock_path, false);
		await this.twine.startClient();

		// then need to listen for incoming messages
		this.listenMessages();
	}
	private async listenMessages() {
		if(this.twine === undefined) throw new Error("twine should not be undefined at this point.")
		for await (const message of this.twine.incomingMessages() ) {
			switch (message.service) {
				case sandboxService:
					this.handleMessage(message);
					break
				case executeService:
					const exec_mod = await import("./ds-exec-service.ts");
					exec_mod.handleMessage(message);
					break;

				case migrateService:
					const migrate_mod = await import('./ds-migrate-service.ts');
					migrate_mod.handleMessage(message);
					break
			
				default:
					message.sendError("service not recognized")
			}
		}
	}
	getTwine() :Twine {
		if(this.twine === undefined) throw new Error("twine should not be undefined at this point.")
		return this.twine;
	}

	private async handleMessage(message:ReceivedMessageI) {
		// For now just pipe straight to route server.
		DsRouteServer.handleServiceMessage(message);
	}
}

const sym = Symbol.for("DropServer DsServices class singleton");
const w = <{[sym]?:DsServices}>window;
if(w[sym] === undefined) w[sym] = new DsServices;

export default w[sym] as DsServices;

