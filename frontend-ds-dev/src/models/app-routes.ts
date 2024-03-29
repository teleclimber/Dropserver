import {computed, reactive} from 'vue';

import twineClient from './twine-client';
import {ReceivedMessageI} from 'twine-web';

// remote service?
const appRoutesService = 16;

// local commands
const allRoutesData = 11
const routesError = 12
const routesDirty = 13
const loadingRoutes = 14

// remote commands
const loadRoutes = 11
const setAutoLoad = 12



export type RoutePath = {
	path: string,
	end: boolean
}
export type RouteAuth = {
	allow: string,	// actually an enum
	permission?: string
}
export type RouteHandler = {
	file?: string,
	function?: string,
	path?: string
}
export type RouteOptions = {
	name?: string,	//used for JS handlers
	path?: string	// used by static DS handler
}
export type RouteConfig = {
	method: string,
	path: RoutePath,
	auth: RouteAuth,
	type: string,	// "function " or "static"
	options: RouteOptions
}

class AppRoutesData {
	routes : RouteConfig[] = [];
	error :string|null = null;	// can we get rid of error, or is it still needed?
	dirty = true;

	_start() {
		twineClient.registerService(appRoutesService, this);
	}

	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case allRoutesData:
				this.loadAllRoutes(m);
				break;
			case routesError:
				this.error = new TextDecoder('utf-8').decode(m.payload);
				this.routes = [];
				this.dirty = false;
				m.sendOK();
				break;
			case routesDirty:
				this.dirty = true;
				m.sendOK();
				break;
			default:
				m.sendError("command not recognized: "+m.command);
		}
	}

	loadAllRoutes(m:ReceivedMessageI) {
		try {
			this.routes = <RouteConfig[]>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
		}
		catch(e) {
			m.sendError("error processing appspace routes "+e);
			console.error(e);
			return;
		}
		this.error = null;
		this.dirty = false;
		m.sendOK();
	}

}

const appRoutesData = reactive(new AppRoutesData());
appRoutesData._start();
export default appRoutesData;