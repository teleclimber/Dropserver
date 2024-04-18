import TwineWebsocketClient, {ReceivedMessageI} from 'twine-web';

interface LocalService {
	handleMessage(m : ReceivedMessageI) :void
}

const ws_addr = (location.protocol==='https:' ? 'wss:' : 'ws:')
				+'//'+location.hostname
				+(location.port ? ':'+location.port : '')
				+'/dropserver-dev/livedata';

class TwineClient {
	_ready = false;
	_ready_resolves :(() => void)[] = [];
	twine:TwineWebsocketClient =  new TwineWebsocketClient(ws_addr);
	local_services : Map<Number,LocalService> = new Map;

	async start() {
		this.serviceDispatcher();
		await this.twine.startClient();
		this._ready = true;
		console.log("twine ready!", this._ready_resolves.length);
		this._ready_resolves.forEach( r => r() )
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

	async ready() :Promise<void> {
		if( this._ready ) return Promise.resolve();
		console.log("twine not ready");
		return new Promise( (resolve:() => void, reject) => {
			this._ready_resolves.push(resolve);
		});
	}
}

const twineClient = new TwineClient;

export default twineClient;