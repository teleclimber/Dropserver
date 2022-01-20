import * as path from "https://deno.land/std@0.106.0/path/mod.ts";
import { Server } from "https://deno.land/std@0.106.0/http/server.ts";
import type {ServerRequest} from "https://deno.land/std@0.106.0/http/server.ts";

import DsServices from './services.ts';
import type AppRoutes from '../approutes.ts';
import type {Context} from '../approutes.ts';

const serverRemoteService = 11;
const serverReadyCommand = 11;

export default class DsRouteServer {
	private server :Server|undefined;

	private stop_resolve :undefined | ((value?: unknown) => void);

	constructor(private services:DsServices, private appRoutes: AppRoutes) {}

	async startServer(sockPath: string) {
		const sockFile = path.join(sockPath, 'server.sock');

		const listener = await Deno.listen({ path: sockFile, transport: "unix" });
		this.server = new Server(listener);

		const listenP = this.listen();	// does this mean errors are uncaught?
		(function() {
			listenP.catch((reason) => {
				console.error("liste rejected: "+reason);
			});
		})()

		const twine = this.services.getTwine();
		const reply = await twine.sendBlock(serverRemoteService, serverReadyCommand, undefined);
		if(!reply.ok) {
			throw reply.error;
		}
		console.log("server started");
	}
	private async listen() {
		if( this.server === undefined ) throw new Error("no server to listen on");
		for await (const request of this.server) {
			this.handleRequest(request);
		}
		console.log("Server has shut down");
		if( this.stop_resolve != undefined ) {
			this.stop_resolve();
		}
	}
	stopServer() {
		return new Promise((resolve, reject) => {
			console.log("Shutting down server");
			if( this.stop_resolve != undefined ) {
				reject("stop already called");
				return;
			}
			this.stop_resolve = resolve;
			console.log("server.close() "+(typeof this.server));
			this.server?.close();
		});
	}

	async handleRequest(request :ServerRequest) {
		const t0 = performance.now();

		if( !this.appRoutes ) {	// should no longer happen
			this.replyError(request, "app ruouter not loaded");
			return;
		}

		const headers = request.headers;
		const matched_route = headers.get("X-Dropserver-Route-ID");
		if( matched_route === null ) {
			this.replyError(request, "X-Dropserver-Route-ID header is null");
			return;
		}
		const route = this.appRoutes.getRouteWithMatch(matched_route);
		if( route === undefined) {
			this.replyError(request, "route id not found");
			return;
		}
		if( !route.handler ) {
			this.replyError(request, "no handler attached to route");
			return;
		}
		if( route.match === undefined ) {
			throw new Error("route returned without match function");
		}

		const req_url_str = headers.get("X-Dropserver-Request-URL");
		if( !req_url_str ) {
			this.replyError(request, "no request url found in headers");
			return;
		}
		const req_url = new URL(req_url_str, "https://appspace/");
		const route_match = route.match(req_url.pathname);
		if( !route_match ) {
			this.replyError(request, "route failed to match in sandbox");
			return;
		}

		const proxyId = headers.get("X-Dropserver-User-ProxyID");

		const ctx :Context = {
			req: request,
			url: req_url,	// request.url is readonly, so we can't set it, so we pass url in context. We could wrap request in Proxy and intercept get url
			params: <Record<string, unknown>>route_match.params,
			proxyId: proxyId
		};

		try {
			await route.handler(ctx);
		}
		catch(e) {
			this.replyError(request, e);
			return;
		}

		const t1 = performance.now();
		console.log(`request took ${t1 - t0} milliseconds.`);
	}

	replyError(req :ServerRequest, message :string) {
		console.error(message);
		req.respond({status: 500, body: message})
	}
}
