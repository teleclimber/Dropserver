import DsServices from "./ds-services.ts";
import type Twine from "./twine/twine.ts";

// TODO  unfinished

// We will have to manage different versions of appspace routes, probably?
// Though I can't recall exactly how we were planning on doing that.
// I think you only get a single version loaded in sandbox at a time?
// Anyways, deal with that later.



const service = 14;

const createCmd = 12;
const deleteCmd = 13;


type Auth = {
	type: string
}

export type FunctionHandler = {	// is that how we want to do this? Would it be better to have several types and overload it?
	type: "function",
	file: string,	// maybe this should be called "module"
	function: string
}
export type FileHandler = {	// is that how we want to do this? Would it be better to have several types and overload it?
	type: "file",
	path: string	// must start with @apspace/... or @app/...
}

export type Route = {
	methods: string[],
	"route-path": string,
	auth: Auth,
	handler: FunctionHandler | FileHandler
}

class Routes {
	
	private twine: Twine;
	constructor() {
		this.twine = DsServices.getTwine();
	}

	// Instead of passing full Twine, 
	// There shoudl be a service-centered wrapper that locks in the service ID
	// and exposes only the minimal surface that a service needs.
	// like sned and sendBlock?


	async createRoute(methods: string[], routePath: string, auth: Auth, handler: FunctionHandler|FileHandler) {

		// TODO: need to validate things on this end.

		const route:Route = {
			methods,
			"route-path": routePath,
			auth,
			handler
		}
		
		const reply = await this.twine.sendBlock(service, createCmd, Routes.makePayload(route));
		if(!reply.ok) {
			console.error("Failed to create route "+routePath+": "+reply.error);
			throw new Error(reply.error);
		}
	}

	async deleteRoute(methods: string[], routePath: string) {
		const route = {
			methods,
			"route-path": routePath
		};

		const reply = await this.twine.sendBlock(service, deleteCmd, Routes.makePayload(route))
		if(!reply.ok) {
			console.error(reply.error);
			throw new Error(reply.error);
		}
	}

	static makePayload(data:object):Uint8Array|undefined {
		if(data === undefined) return undefined;
		return new TextEncoder().encode(JSON.stringify(data));
	}
}

const sym = Symbol.for("DropServer Routes class singleton");
const w = <{[sym]?:Routes}>window;
if(w[sym] === undefined) w[sym] = new Routes;

export default w[sym] as Routes;
