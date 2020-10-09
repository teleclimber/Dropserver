import * as path from "https://deno.land/std/path/mod.ts";
import { Server } from "https://deno.land/std/http/server.ts";
import type {ServerRequest} from "https://deno.land/std/http/server.ts";

import Metadata from "./ds-metadata.ts";
import DsServices from './ds-services.ts';

const serverService = 11;
const serverReadyCommand = 11;

export class DsRouteServer {
	private sock_file:string;
	private server :Server|undefined;

	constructor() {
		this.sock_file = path.join(Metadata.sock_path, 'server.sock');
	}

	async startServer() {
		const listener = await Deno.listen({ path: this.sock_file, transport: "unix" });
		this.server = new Server(listener);

		this.listen();	// does this mean errors are uncaught?

		const twine = DsServices.getTwine();
		const reply = await twine.sendBlock(serverService, serverReadyCommand, undefined)
		if(!reply.ok) {
			throw reply.error;
		}
	}
	private async listen() {
		if( this.server === undefined ) return;

		for await (const request of this.server) {	// does this mean each request is served in sequence?
			const headers = request.headers;
			const mod_file = headers.get('appspace-module');

			const fn = headers.get('appspace-function');

			console.log( 'RUNNER:', mod_file, fn );

			if( mod_file === null ) {
				this.replyError(request, "appspace-module header is null");
				continue;
			}
	
			let mod : any;
			try {
				mod = await import( mod_file );
			}
			catch(e) {
				this.replyError(request, "Failed to import module "+mod_file+" Error: "+e);
			}

			let fnc = mod;
			if( fn ) {
				fnc = mod[fn];
			}

			try {
				fnc(request);
			}
			catch(e) {
				this.replyError(request, e);
			}
		}
	}
	
	async replyError(req :ServerRequest, message :string) {
		console.error(message);
		req.respond({status: 500, body: message})
	}

}

const sym = Symbol.for("DropServer DsRouteServer class singleton");
const w = <{[sym]?:DsRouteServer}>window;
if(w[sym] === undefined) w[sym] = new DsRouteServer;

export default w[sym] as DsRouteServer;