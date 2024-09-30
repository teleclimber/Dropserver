import {match} from "https://deno.land/x/path_to_regexp@v6.2.1/index.ts";
import type {MatchFunction} from "https://deno.land/x/path_to_regexp@v6.2.1/index.ts";

import {RouteType, GetAppRoutesCallback} from 'https://deno.land/x/dropserver_lib_support@v0.2.1/mod.ts';
import type {Handler, Path, Auth} from 'https://deno.land/x/dropserver_lib_support@v0.2.1/mod.ts';

import DsServices from './services/services.ts';

type handlerOpts = {
	name: string
}
type staticOpts = {
	path: string
}

interface RouteBase {
	id: string
	method: string
	path: Path
	auth: Auth
	match?: MatchFunction
}
interface SandboxRoute extends RouteBase {
	type: RouteType.function
	handler: Handler
	opts: handlerOpts
}
interface StaticRoute extends RouteBase {
	type: RouteType.static
	opts: staticOpts
}
type Route = SandboxRoute | StaticRoute

export type RouteExport = {
	id: string,
	method: string,
	path: Path,
	auth: Auth,
	type: RouteType,
	options: staticOpts|handlerOpts
}

/**
 * Class representing a router for application routes.
 */
export default class AppRoutes {
	routes: Map<string,Route> = new Map();
	routes_loaded = false;

	constructor(private services:DsServices) {}

	cb: GetAppRoutesCallback|undefined;
	setCallback(cb:GetAppRoutesCallback) :void {
		if( this.cb !== undefined ) throw new Error("app routes callback already set");
		this.cb = cb;
		// Here we are using this method call as a proxy for determining that the app code is loaded and running.
		// We can therefore set appReady() whcih tells the host that it can proceed fith forwarding requests, etc...
		// Not doubt there are better ways of doing this in the future.
		this.services.appReady();
	}

	loadRoutes() {
		if( this.cb === undefined ) return;
		if( this.routes_loaded ) return;
		const routes = this.cb();
		routes.forEach( r => {
			let stored :Route;
			if(r.type === RouteType.function) {
				stored = {
					id: makeRouteIdentifier(r.method, r.path),
					method: normalizeMethod(r.method),
					path: r.path,
					auth: r.auth,
					type: r.type,
					handler: r.handler,
					opts: {name: r.handlerName}
				};
			}
			else if( r.type === RouteType.static ) {
				stored = {
					id: makeRouteIdentifier(r.method, r.path),
					method: normalizeMethod(r.method),
					path: r.path,
					auth: r.auth,
					type: r.type,
					opts: r.opts
				};
			}
			else {
				//@ts-ignore r.type does not exist here according to TS, but TS doesn't know everything
				throw new Error("Route type not recognized: "+r.type);
			}
			
			this.routes.set(stored.id, stored);
		});
		this.routes_loaded = true;
	}

	exportStack() :RouteExport[] {
		this.loadRoutes();
		const ret :RouteExport[] = [];
		this.routes.forEach( r => {
			ret.push({
				id: r.id,
				method: r.method,
				path: r.path,
				auth: r.auth,
				type: r.type,
				options: r.opts
			});
		});
		return ret;
	}

	getRouteWithMatch(routeId:string) :Route|undefined {
		this.loadRoutes();
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

