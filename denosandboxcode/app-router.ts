import type {ServerRequest} from "https://deno.land/std@0.106.0/http/server.ts";
import {match} from "https://deno.land/x/path_to_regexp@v6.2.0/index.ts";
import type {MatchFunction} from "https://deno.land/x/path_to_regexp@v6.2.0/index.ts";

import {RouteType, GetAppRoutesCallback} from 'https://deno.land/x/dropserver_lib_support@v0.1.0/mod.ts';

// TODO rename file to approutes. (the router is ds-dev-router)

export interface Context {
	req: ServerRequest
	params: Record<string, unknown>
	url: URL
	proxyId: string | null
}

export type Handler = (ctx:Context) => void;

export enum AuthAllow {
	authorized = "authorized",
	public = "public"
}

type Auth = {
	allow: AuthAllow,	//string,	// actually an enum
	permission?: string
}

export type Path = {
	path: string,
	end: boolean
}

type staticOpts = {
	path: string
}

interface Route {
	id: string
	method: string
	path: Path
	auth: Auth
	type: RouteType
	handler?: Handler
	opts?: staticOpts
	match?: MatchFunction
}

export type RouteExport = {
	id: string,
	method: string,
	path: Path,
	auth: Auth,
	type: RouteType,
	options: staticOpts|Record<never,never>
}

/**
 * Class representing a router for application routes.
 */
export default class AppRoutes {
	routes: Map<string,Route> = new Map();

	cb: GetAppRoutesCallback|undefined;
	setCallback(cb:GetAppRoutesCallback) :void {
		if( this.cb !== undefined ) throw new Error("app routes callback already set");
		this.cb = cb;
	}

	loadRoutes() {
		if( this.cb === undefined ) return;
		const routes = this.cb();
		routes.forEach( r => {
			const stored :Route = {
				id: makeRouteIdentifier(r.method, r.path),
				method: normalizeMethod(r.method),
				path: r.path,
				auth: r.auth,
				type: r.type,
				handler: (r.type === RouteType.function ? r.handler : undefined ),
				opts: (r.type === RouteType.static ? r.opts : undefined )
			};
			this.routes.set(stored.id, stored);
		});
	}

	exportStack() :RouteExport[] {
		// iterate over routes
		// and replace known handlers with appropriate data
		const ret :RouteExport[] = [];
		this.routes.forEach( r => {
			let opts:staticOpts|Record<never, never> = {};
			if( r.type === RouteType.static && r.opts) opts = r.opts;
			else if( r.type === RouteType.function && r.handler !== undefined ) opts = {name: r.handler.name};
			else throw new Error("no handler or static opts found in route.");
			ret.push({
				id: r.id,
				method: r.method,
				path: r.path,
				auth: r.auth,
				type: r.type,
				options: opts
			});
		});
		return ret;
	}

	getRouteWithMatch(routeId:string) :Route|undefined {
		const stored = this.routes.get(routeId);
		if( stored === undefined ) return undefined;
		if( stored.match === undefined ) {
			const p = stored.path;
			stored.match = match(p.path, {end:p.end});
		}
		return stored;
	}
}

const methods = ["get", "head", "post", "put", "delete", "connect", "options", "trace", "patch"];
export function normalizeMethod(method:string) :string {
	method = method.toLowerCase();
	if( !methods.includes(method) ) throw new Error("invalid method: "+method);
	return method;
}

export function makeRouteIdentifier(method :string, path:Path) :string {
	return `<${method}><end:${path.end?'true':'false'}>${path.path}`;
}

