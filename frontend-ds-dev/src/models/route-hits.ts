import {reactive} from 'vue';
import twineClient from './twine-client';
import {ReceivedMessageI} from '../twine-ws/twine-common';
import type {RouteConfig, RouteAuth, RouteHandler} from './appspace-routes-data';


const route_commands = {
	hit_event: 11,
};

// Types coming from server:
// These should probably be extracted and reused with all ds frontends

// route hit types:
type Request = {
	url: string,
	method: string
}

type RouteHit = {
	timestamp: Date,
	request: Request,
	route_config: RouteConfig
}

class RouteEvents {
	hit_events :RouteHit[];
	constructor() {
		twineClient.registerService(11, this);
		this.hit_events = reactive([]);
	}
	handleMessage(m:ReceivedMessageI) {
		switch(m.command){
			case route_commands.hit_event:
				this.pushNewHit(m);
			break;
			default:
				m.sendError("unrecognized service");
		}
	}
	pushNewHit(m:ReceivedMessageI) {
		try {
			const hit = <RouteHit>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
			this.hit_events.push(hit);
		}
		catch(e) {
			m.sendError("error processing new hit "+e);
			console.error(e);
			return;
		}

		m.sendOK();
	}
}

const routeEvents = new RouteEvents();

export default routeEvents;
