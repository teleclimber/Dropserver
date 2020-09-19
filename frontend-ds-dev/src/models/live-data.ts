import TwineWebsocketClient from '../twine-ws/index';
import {ReceivedMessageI, ReceivedReplyI} from '../twine-ws/twine-common';
import { reactive } from 'vue';

export const services = Object.freeze({
	routeEvent: 11
});

const ws_addr = 'ws://'+location.hostname+(location.port ? ':'+location.port: '')+'/dropserver-dev/livedata/';

const twine = new TwineWebsocketClient(ws_addr);

const route_commands = {
	hit_event: 11
};

// Types coming from server:
// These should probably be extracted and reused with all ds frontends
type Request = {
	url: string,
	method: string
}
type RouteAuth = {
	type: string
}
type RouteHandler = {
	type: string,
	file?: string,
	function?: string,
	path?: string
}
type RouteConfig = {
	methods: string[],
	path: string,
	auth: RouteAuth,
	handler: RouteHandler
}
type RouteHit = {
	timestamp: Date,
	request: Request,
	route_config: RouteConfig
}

class RouteEvents {
	hit_events :RouteHit[];
	constructor() {
		this.hit_events = reactive([]);
	}
	newMessage(m:ReceivedMessageI) {
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
			console.log(this.hit_events);
		}
		catch(e) {
			m.sendError("error processing new hit "+e);
			console.error(e);
			return;
		}

		m.sendOK();
	}
}

export const routeEvents = new RouteEvents();


async function serviceDispatcher() {
	for await (const m of twine.incomingMessages()) {
		switch(m.service) {
			case services.routeEvent:
				routeEvents.newMessage(m);
			break;
		}
	}
	
}

serviceDispatcher();

twine.startClient();