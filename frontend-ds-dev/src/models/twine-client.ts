import TwineWebsocketClient from '../twine-ws/index';
import {ReceivedMessageI} from '../twine-ws/twine-common';

interface LocalService {
	handleMessage(m : ReceivedMessageI) :void
}

const ws_addr = 'ws://'+location.hostname+(location.port ? ':'+location.port: '')+'/dropserver-dev/livedata/';

class TwineClient {
	twine:TwineWebsocketClient =  new TwineWebsocketClient(ws_addr);
	local_services : Map<Number,LocalService> = new Map;

	start() {
		this.serviceDispatcher();
		this.twine.startClient();
	}

	registerService(serviceID:Number, service :LocalService) {
		if(this.local_services.has(serviceID)) throw new Error("service exists: "+serviceID);
		this.local_services.set(serviceID, service);
	}

	async serviceDispatcher() {
		for await (const m of this.twine.incomingMessages()) {
			const serv = this.local_services.get(m.service);
			if( serv === undefined ) m.sendError("No registered service");
			else serv.handleMessage(m);
		}
	}
}

const twineClient = new TwineClient;

export default twineClient;