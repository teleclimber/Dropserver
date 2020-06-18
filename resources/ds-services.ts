// Consider the possibility that host will push data to a service
// that is not instantiated yet?
// How would that work? Can it even work?

// Well it kind of has. Example: cron.
// - receive message saying run function x at file y

import Twine from "./twine/twine.ts";
import handleExec from "./ds-exec-service.ts";

const sandboxService = 11;
const executeService = 12;

export class DsServices {
	private twine:Twine|undefined;
	constructor() {}

	async initTwine(sock_path: string) {
		if(this.twine !== undefined) throw new Error("Twine already initiated");
		this.twine = new Twine(sock_path, false);
		await this.twine.startClient();

		// then need to listen for incoming messages
		this.listenMessages();
	}
	private async listenMessages() {
		if(this.twine === undefined) throw new Error("twine should not be undefined at this point.")
		for await (const message of this.twine.incomingMessages() ) {
			switch (message.service) {
				case sandboxService:
					throw new Error("not implemented yet");
					// TODO
					break;
				case executeService:
					handleExec(message);
					break;
			
				default:
					message.sendError("service not recognized")
			}
		}
	}
	getTwine() :Twine {
		if(this.twine === undefined) throw new Error("twine should not be undefined at this point.")
		return this.twine;
	}

}

const sym = Symbol.for("DropServer DsServices class singleton");
const w = <{[sym]?:DsServices}>window;
if(w[sym] === undefined) w[sym] = new DsServices;

export default w[sym] as DsServices;

