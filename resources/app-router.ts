import type {ServerRequest} from "https://deno.land/std@0.97.0/http/server.ts";
import {match} from "https://deno.land/x/path_to_regexp@v6.2.0/index.ts";
import type {MatchFunction} from "https://deno.land/x/path_to_regexp@v6.2.0/index.ts";

// Let's make an extremely simple POC router
// That shows that we can create routes programmatically
// in sandbox
// and use them from host side.

export interface Context<P extends object = object> {
	req: ServerRequest
	params: P
	proxy_id: string | null
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

type Route = {
	id: string,
	method: string,
	path: Path,
	auth: Auth,
	handler: Handler,
	match: undefined | MatchFunction
}

enum RouteType {
	function = "function",
	static = "static"
}
export type RouteExport = {
	id: string,
	method: string,
	path: Path,
	auth: Auth,
	type: RouteType,
	options: any
}

// staticOpts contains all the options to pass to the static file handler
type staticOpts = {
	path: string
}

export default class AppRouter {
	stack: Route[] = [];
	dict: Map<string,Route> = new Map();
	static_handlers: Map<Handler,staticOpts> = new Map();

	add(method:string, path:string|Path, auth:Auth, handler:Handler):string {
		method = normalizeMethod(method);
		if( typeof path === 'string' ) path = {path:path, end:true};
		// should normalize path (trailing slash? caps?)
		// .. and ensure there are no dupliacate method + path + end match option
		const r :Route = {
			id: makeRouteIdentifier(method, path),
			method,
			path,
			auth,
			handler,
			match: undefined
		};

		this.stack.push(r);
		this.dict.set(r.id, r);

		return r.id;
	}

	staticFileHandler(opts:staticOpts) :Handler {
		const h = function() {};
		this.static_handlers.set(h, opts);
		return h;
	}

	exportStack() :RouteExport[] {
		// iterate over routes
		// and replace known handlers with appropriate data
		return this.stack.map( r => {
			let type = RouteType.function;
			let opts:any = {name: r.handler.name};
			if( this.static_handlers.has(r.handler) ) {
				type = RouteType.static;
				opts = this.static_handlers.get(r.handler)
			}
			
			return {
				id: r.id,
				method: r.method,
				path: r.path,
				auth: r.auth,
				type: type,
				options: opts
			}
		});
	}

	getRouteWithMatch(route_id:string) :Route|undefined {
		const r = this.dict.get(route_id);
		if( r === undefined ) return undefined;
		if( r.match === undefined ) {
			r.match = match(r.path.path, {end:r.path.end});
		}
		return r;
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

