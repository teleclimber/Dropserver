import Twine from "./twine.ts";
import MigrationService from './migrateservice.ts';
import DsAppService from './appservice.ts';
import DsRouteServer from './routeserver.ts';

import type {ReceivedMessageI} from './twine.ts';

const sandboxService = 11;
const executeService = 12;
const migrateService = 13;
const appService = 14;

const sandboxReadyCommand = 11;

export default class DsServices {
	private twine:Twine|undefined;
	private server :DsRouteServer|undefined;
	private migrationService :MigrationService|undefined;
	private appService :DsAppService|undefined;
	constructor() {}

	setServer(server:DsRouteServer) {
		this.server = server;
	}
	setMigrationService(migrationService:MigrationService) {
		this.migrationService = migrationService;
	}
	setAppService(appService:DsAppService) {
		this.appService = appService;
	}
	async initTwine(sockPath:string) {
		if(this.twine !== undefined) throw new Error("Twine already initiated");
		this.twine = new Twine(sockPath, false);
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
					const exec_mod = await import("./execservice.ts");	// This is not how this should work...
					exec_mod.handleMessage(message);
					break;
				case appService:
					console.log("got appService Message");
					if( this.appService === undefined) message.sendError("appService not present");
					else this.appService.handleMessage(message);
					break;
				case migrateService:
					if( this.migrationService === undefined ) message.sendError("migrationService not present");
					else this.migrationService.handleMessage(message);
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

	private async handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case 13:	// graceful shutdown
				try {
					// All we need to do is stop the route server, and the script should exit.
					if( this.server !== undefined ) await this.server.stopServer();
				}
				catch(e) {
					console.error(e);
					m.sendError(e);
				}
				m.sendOK();
				break;
			default:
				m.sendError("What is this command? "+m.command);
		}
	}

	#server_ready = false;
	serverReady() {
		if( this.#server_ready ) return;
		this.#server_ready = true;
		this.#sendReady();
	}
	#app_ready = false;
	appReady() {
		if( this.#app_ready ) return;
		this.#app_ready = true;
		this.#sendReady();
	}
	async #sendReady() {
		if(this.#server_ready && this.#app_ready ) {
			const twine = this.getTwine();
			const reply = await twine.sendBlock(sandboxService, sandboxReadyCommand, undefined);
			if(!reply.ok) {
				throw reply.error;
			}
		}
	}
}
