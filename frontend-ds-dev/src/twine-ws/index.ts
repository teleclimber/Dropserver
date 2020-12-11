import { services, commands, encodeMessageMeta, BytesToMessages, MessageBuffer, MessageRegistry, messageMeta, Message, Msg, SentMessageI, ReceivedMessageI, ReceivedReplyI } from './twine-common';

export default class TwineWebsocketClient {
	
	private ws: WebSocket | undefined
	private msgReg:     MessageRegistry
 
	private bytes2messages : BytesToMessages = new BytesToMessages;

	private incomingQueue: MessageBuffer = new MessageBuffer;

	private _graceful: boolean = false;
	// private connClosed: boolean;

	constructor(private address:string) {
		this.msgReg = new MessageRegistry(1, 127);
	}
	async startClient() {
		return new Promise<void>((resolve, reject) => {
			this.ws = new WebSocket(this.address);
			this.ws.onerror = (err) => {
				console.error('ws error', err);
			}
			this.ws.binaryType = "arraybuffer";

			this.ws.onopen = async () => {
				if( this.ws === undefined ) throw new Error("ws is undefined");
				this.ws.onmessage = (event) => {
					this.bytes2messages.push(new Uint8Array(event.data));
				}
				this.receive();

				try {
					await this.sendHi();
				}
				catch(e) {
					reject(e);
					return
				}

				resolve();
			}
		});
	}
	async * incomingMessages() : AsyncGenerator<ReceivedMessageI, void, void> {
		for await (const m of this.incomingQueue) {
			if( m === undefined ) break;	// shouldn't have to do that, but TS Gods need their sacrifices.
			yield m;
		}
	}

	// receive stuff 
	async receive() : Promise<void> {
		for await ( const raw of this.bytes2messages ) {

			if( raw.service === services.protocol ) {
				this.handleProtocolCmd(raw);
				continue;
			}

			if( raw.service == services.refRequest ) {
				const refMsgData = this.msgReg.getOpenMessage(raw.refMsgID);
				const msgData = this.msgReg.registerMessage(raw);
				const message = this.makeMessage(raw, msgData);
				message.service = refMsgData.service;

				// TODO: should try-catch this, and send Error if err
				setTimeout(() => refMsgData.pushRefMessage(message), 0);
			}
			else if( raw.service === services.reply ) {
				const msgData = this.msgReg.closeMessage(raw.msgID); // since this is a reply, this was an **outgoing** message id
				const message = this.makeMessage(raw, undefined); // don't pass ref msg since it's a reply
				message.service = msgData.service;
				setTimeout(() => msgData.setReply(message), 0);
			}
			else if( raw.service === services.close ) { // handles OK and Error messages
				const msgData = this.msgReg.getMessageData(raw.msgID)
				if( this.msgReg.msgIDIsLocal(raw.msgID) ) { // This is a reply to a sent message. It should be open.
					if( msgData.closed ) {
						throw new Error("Message is closed")
					}
				}
				else {
					if( !msgData.closed ) {
						throw new Error("Received reply acknowledgement on open message")
					}
				}
				this.msgReg.unregisterMessage(raw.msgID)
				const message = this.makeMessage(raw, undefined) // we pass a message, but do not connect any ref msg data because the message is at end of life
				message.service = msgData.service;
				setTimeout(() => msgData.setReply(message), 0);
			} else {
				// Brand new message, check we're not graceful, then register message id
				const msgData = this.msgReg.registerMessage(raw) // this is **incoming** message, maybe check it's in the right range
	
				if( raw.service != services.protocol ) {
					const message = this.makeMessage(raw, msgData)
					
					if( this._graceful ) {
						// message received while we are terminating.
						// Can happen in normal course of things.
						// TODO: in go we have this in a goroutine so it sends async.
						message.sendError("terminating")
					} else {
						setTimeout(() => this.incomingQueue.push(message), 0);
					}
				}
			}
		}
	}

	// SENDS...
	async send(service: number, cmd: number, payload: Uint8Array|undefined) : Promise<SentMessageI> {
		const newMsgID = this.msgReg.newMessage(service) // should maybe return an error in case no message ids left
	
		const m :SentMessageI = new Message({
			service: service,
			command: cmd,
			msgID: newMsgID,
			refMsgID: 0,
			payload: payload
		}, this)
		m.msg = this.msgReg.getMessageData(newMsgID)
			
		await this._send(newMsgID, 0, service, cmd, payload);
	
		return m;
	}
	// external commands
	async sendBlock(service: number, cmd: number, payload: Uint8Array|undefined) : Promise<ReceivedReplyI> {
		const newMsgID = this.msgReg.newMessage(service) // should maybe return an error in case no message ids left
	
		await this._send(newMsgID, 0, service, cmd, payload);

		const msg = this.msgReg.getMessageData(newMsgID);

		const reply = await msg.waitReply();
		//if(reply.error) throw reply.error;	// maybe don't thorw? maybe the message consumer should check the error?

		return reply;
	}

	async reply(msgID : number, cmd :number, payload: Uint8Array|undefined) {
		this.msgReg.assertMsgIDRemote(msgID);
			
		const msgData = this.msgReg.closeMessage(msgID);
		
		// check session is same and still open?
		// check msgID is still open (it should be if this is the reply, but need to be sure they only reply once)
	
		await this._send(msgID, 0, services.reply, cmd, payload);

		const ack = await msgData.waitReply()
		if( ack.error ) throw ack.error;	// hmm, is throwing really what we want to do here?	
	}

	async replyOKClose(msgID :number) {
		this.msgReg.assertMsgIDRemote(msgID);
			
		const msgData = this.msgReg.getMessageData(msgID);

		if( this.msgReg.msgIDIsLocal(msgID) ) {
			if( !msgData.closed ) {
				throw new Error("expected to send OK on closed message");
			}
		} else {
			if( msgData.closed ) {
				throw new Error("msg ID is closed")
			}
		}
	
		await this._send(msgID, 0, services.close, commands.ok, undefined) // cmd is 0 on ok close?
	
		this.msgReg.unregisterMessage(msgID);
	}

	async replyErrorClose(msgID :number, err_str:string) {
		this.msgReg.assertMsgIDRemote(msgID);
			
		const msgData = this.msgReg.getMessageData(msgID);

		if( this.msgReg.msgIDIsLocal(msgID) ) {
			if( !msgData.closed ) {
				throw new Error("expected to send OK on closed message");
			}
		} else {
			if( msgData.closed ) {
				throw new Error("msg ID is closed")
			}
		}
	
		await this._send(msgID, 0, services.close, commands.error, new TextEncoder().encode(err_str)) // cmd is 0 on ok close/err?
	
		this.msgReg.unregisterMessage(msgID);
	}

	// RefRequest sneds a new message with a reference to an open message
	async refRequest(refID :number, cmd :number, payload :Uint8Array|undefined) :Promise<SentMessageI> {
		this.msgReg.assertMsgIDRange(refID);
		
		const refMsgData = this.msgReg.getOpenMessage(refID);
		
		if( refMsgData.closed ) {
			throw new Error("Message ID is closed");
		}

		const newMsgID = this.msgReg.newMessage(refMsgData.service)
		
		const m:SentMessageI = new Message({
			command:  cmd,
			service:  services.refRequest,
			msgID:    newMsgID,
			refMsgID: refID,
			payload
		},this);
		m.msg = this.msgReg.getMessageData(newMsgID);

		await this._send(newMsgID, refID, services.refRequest, cmd, payload)

		return m
	}

	async refRequestBlock(refID :number, cmd :number, payload :Uint8Array) :Promise<ReceivedReplyI> {
		this.msgReg.assertMsgIDRange(refID);
		
		const refMsgData = this.msgReg.getOpenMessage(refID);
		
		if( refMsgData.closed ) {
			throw new Error("Message ID is closed");
		}

		const newMsgID = this.msgReg.newMessage(refMsgData.service)

		await this._send(newMsgID, refID, services.refRequest, cmd, payload)
		
		const msg = this.msgReg.getMessageData(newMsgID)

		return msg.waitReply();
	}

	async sendMsgClosed(msgID :number) { // do we send any kind of error?
		await this._send(msgID, 0, services.protocol, commands.msgError, undefined)
	}
	
	// func (t *Twine) sendMsgError(msgID uint8) { // do we send any kind of error?
	// 	t.send(msgID, protocolService, uint8(protocolMsgError), nil)
	// }

	// internal sends:
	private async sendHi() {
		await this.sendBlock(services.protocol, commands.hi, undefined);
	}

	// in websockets pings are handled by the browser?
	// private async sendPing() {
	// 	const reply = await this.sendBlock(protocolService, protocolPing, undefined)
		
	// 	if( reply.command !== protocolPong ) {
	// 		throw new Error("response to Ping was not Pong");
	// 	}
	
	// 	await reply.sendOK()
	// }

	private async _send(msgID: number, refMsgID: number, service: number, cmd: number, payload: Uint8Array|undefined){
		if( !this.ws ) throw new Error("can't send: ws undefined");

		const meta_data = encodeMessageMeta(msgID, refMsgID, service, cmd, payload);

		let send_data = meta_data;
		if( payload ) {
			send_data = new Uint8Array(meta_data.length + payload.length);
			send_data.set(meta_data);
			send_data.set(payload, meta_data.length);
		}

		this.ws.send(send_data);
	}

	makeMessage(raw :messageMeta, ref: Msg|undefined) :Message {
		const m = new Message(raw, this);
		m.msg = ref;
		return m;
	}

	async handleProtocolCmd(raw: messageMeta) {
		let newMsg: Msg, message: Message;
		switch(raw.command) {
		// 1 is "hi", handled separately
		case commands.graceful:
			newMsg = this.msgReg.registerMessage(raw);
			message = this.makeMessage(raw, newMsg);
			this.receivedGraceful(message);
			break;
	
		case commands.ping:
			newMsg = this.msgReg.registerMessage(raw);
			message = this.makeMessage(raw, newMsg);
			await message.reply(commands.pong, undefined);
			break;
	
		default:
			throw new Error("what is this protocol command? "+raw.command);
		}
	
		return
	}

	async graceful() {
		const reply = await this.sendBlock(services.protocol, commands.graceful, undefined);
		if( reply.error ) throw reply.error;

		await this.doGraceful()
	}

	async receivedGraceful(message :ReceivedMessageI) {
		message.sendOK();
		await this.doGraceful();
	}
	async doGraceful() {
		if( this._graceful ) throw new Error("Already in graceful shutdown");
		this._graceful = true;
		
		await this.msgReg.waitAllUnregistered();

		await this.close();
	}

	async close() {
		if( !this.ws ) return;

		// close other stuf...

		this.ws.close();
		this.ws = undefined;

		this.incomingQueue.stop();
	}
}








