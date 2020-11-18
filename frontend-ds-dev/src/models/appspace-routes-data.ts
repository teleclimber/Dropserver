import {computed, reactive} from 'vue';

import twineClient from './twine-client';

import {ReceivedMessageI} from '../twine-ws/twine-common';

const appspaceRoutesService = 16;

export type RouteAuth = {
	type: string
}
export type RouteHandler = {
	type: string,
	file?: string,
	function?: string,
	path?: string
}
export type RouteConfig = {
	methods: string[],
	path: string,
	auth: RouteAuth,
	handler: RouteHandler
}

type RoutesData = {
	path: string,
	routes: RouteConfig[]
}

class AppspaceRoutesData {
	routes : RouteConfig[] = [];

	_start() {
		twineClient.registerService(appspaceRoutesService, this);
	}

	handleMessage(m:ReceivedMessageI) {
		switch (m.command) {
			case 11:
				this.loadAllRoutes(m);
				break;
		
			case 12:
				this.patchRoutes(m);
				break;
			default:
				m.sendError("command not recognized: "+m.command);
		}
	}

	loadAllRoutes(m:ReceivedMessageI) {
		try {
			const data = <RoutesData>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			this.routes = data.routes;
		}
		catch(e) {
			m.sendError("error processing appspace routes "+e);
			console.error(e);
			return;
		}
		console.log('all routes', this.routes);
		m.sendOK();
	}
	patchRoutes(m:ReceivedMessageI) {
		let data :RoutesData;
		try {
			data = <RoutesData>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
		}
		catch(e) {
			m.sendError("error processing appspace routes "+e);
			console.error(e);
			return;
		}
		m.sendOK();

		const p = data.path;
		this.routes = this.routes.filter( (r:RouteConfig) => r.path !== p );
		
		if( data.routes ) this.routes.push(...data.routes);

		console.log( 'patched routes: ', this.routes);
	}

	get sorted() {
		return computed(()=>{
			return this.routes.slice().sort( (a,b)=> {
				return a.path.localeCompare(b.path);
			});
		})
	}
}

const appspaceRoutesData = reactive(new AppspaceRoutesData());
appspaceRoutesData._start();
export default appspaceRoutesData;