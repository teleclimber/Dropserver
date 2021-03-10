/////////////////////////////////////////////////
// First pass at Twine.ts:

// reserved service IDs:
const
	protocolService   = 1,
	refRequestService = 4, // New mesage with a reference to open message
	replyService      = 5, // reply to a message
	closeService      = 6; // reply with standard OK/Err, or acknowledge reply

// reserved command ids:
const
	protocolHi    = 1,
	protocolOK    = 2,
	protocolError = 3;

// Used by protocol:
const
	protocolPing     = 4,
	protocolPong     = 5,
	protocolMsgError = 6, // error at protocol/transport level? Is ther anything there?
	protocolGraceful = 7; // Would like to shutdown


type messageMeta = {
	service: number,
	command: number,
	msgID: number,
	refMsgID: number,
	payload:  Uint8Array | undefined
};


export default class Twine {

	private pre_conn_messages: Uint8Array[] = [];
	
	//private ln         *net.Listener // consider making that a ReadWriter so we can adapt anything there.
	private server_listener: Deno.Listener | undefined
	private conn: Deno.Conn | undefined
	private msgReg:     MessageRegistry
 
	// private ReadyChan   chan struct{}
	// private MessageChan chan ReceivedMessageI
	// private ErrorChan   chan error

	private bytes2messages : BytesToMessages = new BytesToMessages;

	private incomingQueue: MessageBuffer = new MessageBuffer;

	private _graceful: boolean = false;
	// private connClosed: boolean;

	constructor(private sock_path: string, private isServer :boolean) {	//doesn't make sense to pass isServer. This is determined by startServer or startClient.
		if( this.isServer ) {
			this.msgReg = new MessageRegistry(128, 255);
		}
		else {
			this.msgReg = new MessageRegistry(1, 127);
		}

		// messageChan... I think this is an itera** thing.

	}
	// this should really just be "start", and branch off based on isServer
	async startServer() {
		if( !this.isServer ) throw new Error("starting server on non-server");

		this.server_listener = await Deno.listen({path: this.sock_path, transport:"unix"});
		this.conn = await this.server_listener.accept();

		// shouldw e not first receive message HI?
		const first_chunk = new Uint8Array(10);
		const num = await this.conn.read(first_chunk);
		const raw = this.decodeMessage(first_chunk);	// this is somewhat dangerous given we know chunks can be merged.
		if( raw.service !== protocolService || raw.command !== protocolHi ) {
			throw new Error("first message not hi");
		}
		this._send(raw.msgID,0, closeService, protocolOK, undefined);

		this.receive();

		this.flushPreConnMessages();
	}
	async startClient() {	// maybe this is a generator function that yields incoming messages?
		if( this.isServer ) throw new Error("starting client on server");

		// warning unstable API in Deno. Use "--unstable"
		this.conn = await Deno.connect({ path: this.sock_path, transport: "unix" });

		this.receive();

		await this.sendHi();
		// catch errors with this

		this.flushPreConnMessages();
	}
	async flushPreConnMessages() {
		this.pre_conn_messages.forEach(m => {
			if( this.conn === undefined ) throw new Error("expected a conn to flush messages to");
			this.conn.write(m);
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
		(async() => {
			if( this.conn === undefined ) throw new Error("conn is undefined");
			try {
				for await (const chunk of Deno.iter(this.conn)) {
					this.bytes2messages.push(chunk);
				}
			}
			catch(e) {
				if (!(e instanceof Deno.errors.BadResource)) { // could be more subtle by checking if we intended to close the conn with if( this.stop ) or whtaevr
					throw e; 
				}
			}
		})();
		for await ( const raw of this.bytes2messages ) {

			if( raw.service === protocolService ) {
				this.handleProtocolCmd(raw);
				continue;
			}

			if( raw.service == refRequestService ) {
				const refMsgData = this.msgReg.getOpenMessage(raw.refMsgID);
				const msgData = this.msgReg.registerMessage(raw);
				const message = this.makeMessage(raw, msgData);
				message.service = refMsgData.service;

				// TODO: should try-catch this, and send Error if err
				setTimeout(() => refMsgData.pushRefMessage(message), 0);
			}
			else if( raw.service === replyService ) {
				const msgData = this.msgReg.closeMessage(raw.msgID); // since this is a reply, this was an **outgoing** message id
				const message = this.makeMessage(raw, undefined); // don't pass ref msg since it's a reply
				message.service = msgData.service;
				setTimeout(() => msgData.setReply(message), 0);
			}
			else if( raw.service === closeService ) { // handles OK and Error messages
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
	
				if( raw.service != protocolService ) {
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

	decodeMessage(chunk: Uint8Array) : messageMeta {	// duplicates decode in BytesToMessages.
		const view = new DataView(chunk.buffer);

		const msg :messageMeta = {
			service: view.getUint8(0),
			command: view.getUint8(1),
			msgID: view.getUint8(2),
			refMsgID: 0,
			payload: undefined
		}
		
		let offset = 3;

		if( msg.service === refRequestService ) {
			msg.refMsgID = view.getUint8(offset);
			++offset;
		}

		let pSize = view.getUint16(offset);
		offset += 2;
		if( pSize === 0xff ) {
			pSize = view.getUint32(offset);
			offset += 4;
		}

		if( chunk.length < offset+pSize ) {
			throw new Error("didn't get the full message with payload");
		}

		msg.payload = chunk.slice(offset, offset+pSize);

		return msg;
	}
	private static encodeMessageMeta(msgID: number, refMsgID: number, service: number, cmd: number, payload: Uint8Array|undefined) : Uint8Array {
		if( service < 1 || service > 0xff ) {
			throw new Error("service id is out of bounds");
		}
		if( cmd < 0 || cmd > 0xff ) {
			throw new Error("cmd id is out of bounds");
		}
		if( msgID < 0 || msgID > 0xff ) { // allow zero to send Bye		
			throw new Error("send: message id is out of bounds");
		}

		let buf = new ArrayBuffer(10);
		let view = new DataView(buf);

		view.setUint8(0, service);
		view.setUint8(1, cmd);
		view.setUint8(2, msgID);
		let cur_offset = 3;

		if( service === refRequestService ) {
			if( refMsgID < 1 || refMsgID > 0xff ) {
				throw new Error("send: reference message id is out of bounds");
			}
			view.setUint8(3, refMsgID);
			++cur_offset;
		}

		let pSize = payload === undefined ? 0 : payload.length;
		if( pSize >= 0xff ) {
			view.setUint16(cur_offset, 0xff);
			cur_offset += 2;
			view.setUint32(cur_offset, pSize);
			cur_offset += 4;
		}
		else {
			view.setUint16(cur_offset, pSize);
			cur_offset += 2;
		}

		return new Uint8Array(buf, 0, cur_offset);
	}

	// read low level functions?

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
	
		await this._send(msgID, 0, replyService, cmd, payload);

		const ack = await msgData.waitReply()
		if( ack.error ) throw ack.error;	// hmm, is throwing really what we want to do here?	
	}

	async replyOKClose(msgID :number) {
		this.msgReg.assertMsgIDRange(msgID);
			
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
	
		await this._send(msgID, 0, closeService, protocolOK, undefined) // cmd is 0 on ok close?
	
		this.msgReg.unregisterMessage(msgID);
	}

	async replyErrorClose(msgID :number, err_str:string) {
		this.msgReg.assertMsgIDRange(msgID);
			
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
	
		await this._send(msgID, 0, closeService, protocolError, new TextEncoder().encode(err_str)) // cmd is 0 on ok close/err?
	
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
			service:  refRequestService,
			msgID:    newMsgID,
			refMsgID: refID,
			payload
		},this);
		m.msg = this.msgReg.getMessageData(newMsgID);

		await this._send(newMsgID, refID, refRequestService, cmd, payload)

		return m
	}

	async refRequestBlock(refID :number, cmd :number, payload :Uint8Array) :Promise<ReceivedReplyI> {
		this.msgReg.assertMsgIDRange(refID);
		
		const refMsgData = this.msgReg.getOpenMessage(refID);
		
		if( refMsgData.closed ) {
			throw new Error("Message ID is closed");
		}

		const newMsgID = this.msgReg.newMessage(refMsgData.service)

		await this._send(newMsgID, refID, refRequestService, cmd, payload)
		
		const msg = this.msgReg.getMessageData(newMsgID)

		return msg.waitReply();
	}

	async sendMsgClosed(msgID :number) { // do we send any kind of error?
		await this._send(msgID, 0, protocolService, protocolMsgError, undefined)
	}
	
	// func (t *Twine) sendMsgError(msgID uint8) { // do we send any kind of error?
	// 	t.send(msgID, protocolService, uint8(protocolMsgError), nil)
	// }

	// internal sends:
	private async sendHi() {
		await this.sendBlock(protocolService, protocolHi, undefined);
	}

	private async sendPing() {
		const reply = await this.sendBlock(protocolService, protocolPing, undefined)
		
		if( reply.command !== protocolPong ) {
			throw new Error("response to Ping was not Pong");
		}
	
		await reply.sendOK()
	}

	// ooook. This is where we need to smarten up
	private async _send(msgID: number, refMsgID: number, service: number, cmd: number, payload: Uint8Array|undefined){
		const meta_data = Twine.encodeMessageMeta(msgID, refMsgID, service, cmd, payload);
		let full = meta_data;
		if( payload ) {
			full = new Uint8Array(meta_data.length+ (payload?payload.length:0) );
			full.set(meta_data, 0);
			full.set(payload, meta_data.length)
		}

		if( this.conn ) {
			await this.conn.write(full);
		}
		else {
			this.pre_conn_messages.push(full);
		}
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
		case protocolGraceful:
			newMsg = this.msgReg.registerMessage(raw);
			message = this.makeMessage(raw, newMsg);
			this.receivedGraceful(message);
			break;
	
		case protocolPing:
			newMsg = this.msgReg.registerMessage(raw);
			message = this.makeMessage(raw, newMsg);
			await message.reply(protocolPong, undefined);
			break;
	
		default:
			throw new Error("what is this protocol command?");
		}
	
		return
	}

	async graceful() {
		const reply = await this.sendBlock(protocolService, protocolGraceful, undefined);
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
		if( !this.conn ) return;

		// close other stuf...

		this.conn.close();
		this.conn = undefined;

		if( this.server_listener !== undefined ) {
			this.server_listener.close();
			this.server_listener = undefined;
		}
		this.incomingQueue.stop();
	}
}

class Msg {
	
	private _closed: boolean = false;

	private _reply : ReceivedReplyI | undefined;
	private _resolveReply: ((m:ReceivedReplyI) => void) | undefined;

	private incomingQueue: MessageBuffer | undefined;

	constructor( private _service: number) {

	}

	get service() {
		return this._service;
	}

	close() {
		this._closed = true;
		if( this.incomingQueue !== undefined ) {
			this.incomingQueue.stop();
		}
	}
	get closed() {
		return this._closed;
	}

	setReply(m: ReceivedReplyI) {
		if( this._reply !== undefined ) throw new Error("setting multiple replies");
		this._reply = m;
		if( this._resolveReply !== undefined ) {
			this._resolveReply(m);
		}
	}
	async waitReply() : Promise<ReceivedReplyI> {
		return new Promise( (resolve, reject) => {
			if( this._reply !== undefined ) resolve(this._reply);
			else {
				if( this._resolveReply !== undefined ) {
					reject("multiple waits for a reply");
					return;
				}
				this._resolveReply = resolve;
			}
		});
	}

	pushRefMessage(m: Message){
		if( this.incomingQueue === undefined ) {	// not sure that's right. If there are no takes for the messages, we should reply with error.
			this.incomingQueue = new MessageBuffer;
		}
		this.incomingQueue.push(m);
	}
	async * incomingMessages() : AsyncGenerator<ReceivedMessageI, void, void> {
		if( this.incomingQueue === undefined ) {
			this.incomingQueue = new MessageBuffer;
		}
		for await (const m of this.incomingQueue) {
			if( m === undefined ) break;	// shouldn't have to do that, but TS Gods need their sacrifices.
			yield m;
		}
	}
}

class MessageRegistry {
	//messagesMux sync.Mutex
	messages = new Map<number, Msg>();
	nextID:      number;
	resolveAllUnregistered: (() => void) | undefined;

	constructor(private firstMsgID: number, private lastMsgID: number) {
		this.nextID = firstMsgID;
	}

	incrementNextID() {
		this.nextID++
		if(this.nextID > this.lastMsgID) {
			this.nextID = this.firstMsgID;
		}
	}
	assertMsgIDRange(msgID: number) {
		if( msgID == 0 || msgID > 0xff ) {
			throw new Error("message ID out of range");
		}
	}
	msgIDIsLocal(msgID: number) :boolean {
		return msgID >= this.firstMsgID && msgID <= this.lastMsgID
	}
	assertMsgIDRemote(msgID: number) {
		this.assertMsgIDRange(msgID);
		if( this.msgIDIsLocal(msgID) ) {
			throw new Error("message ID in wrong range");
		}
	}

	newMessage(service: number) : number {
		let has;
		do {
			this.incrementNextID()
			has = this.messages.has(this.nextID);
		} while(has);
	
		let newID = this.nextID;
		let newMsg = new Msg(service);
	
		this.messages.set(newID,newMsg);
	
		this.incrementNextID()
	
		return newID;
	}
	registerMessage(raw: messageMeta) : Msg {
		if( this.msgIDIsLocal(raw.msgID) ) {
			throw new Error("message id is local, expected remote")
		}
	
		if( this.messages.has(raw.msgID) ) {
			throw new Error("Message id already registered");
		}
	
		let newMsg = new Msg(raw.service);
		this.messages.set(raw.msgID, newMsg);
	
		return newMsg;
	}

	closeMessage(msgID : number) : Msg {
		let msgData = this.messages.get(msgID);
		if( !msgData ) {
			throw new Error("message ID not found");
		}
		if( msgData.closed ) {
			throw new Error("message was already closed");
		}
		msgData.close();

		return msgData;
	}
	unregisterMessage(msgID: number) {
		let msgData = this.messages.get(msgID)
		if( !msgData ) {
			throw new Error("message ID is not registered");
		}

		msgData.close();

		this.messages.delete(msgID);

		if( this.resolveAllUnregistered !== undefined && this.messages.size === 0 ) {
			this.resolveAllUnregistered();
			this.resolveAllUnregistered = undefined;
		}
	}
	getOpenMessage(msgID: number) : Msg {
		let msgData = this.messages.get(msgID);
		if( !msgData ) {
			throw new Error("message ID not found: %v");
		}
	
		if( msgData.closed ) {
			throw new Error("message ID is closed");
		}
	
		return msgData;
	}
	getMessageData(msgID:number) : Msg {
		let msgData = this.messages.get(msgID)
		if( !msgData ) {
			throw new Error("message ID not found");
		}
	
		return msgData;
	}
	
	async waitAllUnregistered() {
		// should also prevent further messages from registering at this point
		if(this.resolveAllUnregistered !== undefined) throw new Error("alredy waiting for all unregistered");
		if(this.messages.size === 0) return;
		return new Promise( (resolve, reject) => {
			this.resolveAllUnregistered = resolve;
		});
	}
}

// Messages:
interface MessageI {
	msgID: number,
	refMsgID: number,
	service: number,
	command: number,
	payload: Uint8Array | undefined,
	msg: Msg | undefined,
}
interface MessageGetReplyI {
	waitReply() :Promise<ReceivedReplyI>
}

interface MessageReplyOKErrI {
	sendOK() : void,
	sendError(err: string) : void
}

interface MessageReplierI {
	reply(cmd: number, payload: Uint8Array|undefined) : void
}

interface MessageReceivedOKI {
	ok: boolean,
	error: string | undefined
}

interface MessageRefererI {
	refSend(command: number, payload: Uint8Array|undefined) : Promise<SentMessageI>,
	refSendBlock(command: number, payload: Uint8Array|undefined) : Promise<ReceivedReplyI>
	incomingMessages(): AsyncGenerator<ReceivedMessageI, void, void>
}

export interface SentMessageI extends MessageI, MessageGetReplyI, MessageRefererI {}
export interface ReceivedMessageI extends MessageI, MessageReplierI, MessageReplyOKErrI, MessageRefererI {}
export interface ReceivedReplyI extends MessageI, MessageReceivedOKI, MessageReplyOKErrI {}

export class Message {
	msgID: number
	refMsgID: number
	service: number
	command: number
	payload: Uint8Array| undefined
	msg: Msg | undefined

	constructor(msg_meta: messageMeta, private t:Twine) {
		this.msgID = msg_meta.msgID
		this.refMsgID = msg_meta.refMsgID
		this.service = msg_meta.service
		this.command = msg_meta.command
		this.payload = msg_meta.payload
	}

	async waitReply() :Promise<ReceivedReplyI> {
		if( this.msg === undefined ) throw new Error("Mising stashed message object");
		return this.msg.waitReply();
	}

	async sendOK() {
		await this.t.replyOKClose(this.msgID);
	}
	async sendError(errStr: string) {
		await this.t.replyErrorClose(this.msgID, errStr);
	}
	async reply(cmd: number, payload: Uint8Array|undefined) {
		await this.t.reply(this.msgID, cmd, payload);
		
	}

	get ok(): boolean {
		return this.command === protocolOK;
	}
	
	// Error returns an error if the reply was an error
	get error(): string| undefined {
		if( this.command === protocolError) {
			if( this.payload !== undefined ) {
				return new TextDecoder("utf-8").decode(this.payload);
			}
			return "Error sent with no description";
		}
		return undefined;
	}

	async refSend(cmd: number, payload: Uint8Array|undefined) : Promise<SentMessageI> {
		return await this.t.refRequest(this.msgID, cmd, payload);
	}
	
	// RefSendBlock sends a new mssage referencing anexisting one,
	// and returns with the response or an error
	async refSendBlock(cmd : number, payload : Uint8Array|undefined) : Promise<ReceivedReplyI> {
		let sent = await this.t.refRequest(this.msgID, cmd, payload);
			
		return await sent.waitReply()
	}

	async * incomingMessages(): AsyncGenerator<ReceivedMessageI, void, void> {
		if( this.msg === undefined ) throw new Error("Mising stashed message object");
		for await (const m of this.msg.incomingMessages() ) {
			yield m;
		}
	}
}

// going to try a message buffer
// TODO: need a way to stop cleanly.
export class MessageBuffer {
	private buf: ReceivedMessageI[] | undefined;
	private nextRead: number = 0;
	private _stop = false;
	private resolveMessage: ((r:{value?:ReceivedMessageI, done: boolean}) => void) | undefined;

	constructor() {}

	push(m: ReceivedMessageI) {
		if( this.resolveMessage !== undefined) {
			this.resolveMessage({done: false, value:m});
			this.resolveMessage = undefined;
		}
		else {
			if( this.buf === undefined ) this.buf = [];
			else if( this.nextRead > 10 ) {
				this.buf = this.buf.slice(this.nextRead);
				this.nextRead = 0;
			}
			this.buf.push(m);
		}
	}
	async next(): Promise<{value?:ReceivedMessageI, done: boolean}> {
		if( this._stop ) {
			return { done: true };
		}
		else if( this.buf !== undefined && this.nextRead < this.buf.length ) {
			++this.nextRead;
			return { value: this.buf[this.nextRead-1], done: false };
		}
		else {
			return new Promise( (resolve, reject) => {
				if( this.resolveMessage !== undefined ) {
					throw new Error("expected resolve to be undefined?");
					// reject instead
				}
				this.resolveMessage = resolve;
			});
		}
	}
	[Symbol.asyncIterator]() { return this; }

	stop() {
		this._stop = true;
		if( this.resolveMessage !== undefined ) {
			this.resolveMessage({done: true})
		}
	}
}

// Now need something that takes bytes and pushes out messageMeta
// Do a circular buffer of chunks. This prevents a lot of copying of data
// you just need to keep track of the current chunk_i, and byte_i

type incomingMessage = {
	msg: messageMeta,
	meta_length:number,
	payload_remaining:number
}

// exported only for testing purposes.
export class BytesToMessages {
	private max_size: number
	private buf : Uint8Array[];
	
	private start: number = 0;	// first chunk of new data
	private cur_size: number = 0;	// last chunk of new data
	private byte_offset: number = 0; //position within chunk of next new data

	private cur_message: incomingMessage | undefined;

	private resolveMessage: ((r:{value:messageMeta, done: boolean}) => void) | undefined;

	private is_stopped : boolean = false;

	constructor(){
		this.max_size = 100;	// 100 chunks
		this.buf = [];
	}

	push(chunk: Uint8Array) {
		const next_i = this.nextWriteI();
		this.buf[next_i] = chunk;
		++this.cur_size;

		if( this.cur_message === undefined ) {
			this.decodeNext();
		}
		else {
			this.pushChunkOnPayload();
		}
	}
	private nextWriteI() {
		if( this.cur_size >= this.max_size ) throw new Error("buffer full");
		return (this.start + this.cur_size) % this.max_size;
	}
	private nextReadI() {
		return (this.start + 1) % this.max_size;
	}
	private advanceChunk() {
		if( this.byte_offset === this.buf[this.start].byteLength ) {
			this.start = this.nextReadI();
			this.byte_offset = 0;
			--this.cur_size;
		}
	}
	
	async next(): Promise<{value:messageMeta, done: boolean}> {
		if( this.is_stopped ) {
			return { done: true, value:{msgID:0, refMsgID:0, command:0, service:0, payload: undefined} };
		}
		else if( this.cur_message !== undefined && this.cur_message.payload_remaining === 0 ) {
			const msg = this.cur_message.msg;
			this.cur_message = undefined;
			setTimeout(() => this.decodeNext(), 0);
			return { value: msg, done: false};
		}
		else {
			return new Promise( (resolve, reject) => {
				if( this.resolveMessage !== undefined ) {
					throw new Error("expected resolve to be undefined?");
				}
				this.resolveMessage = resolve;
			});
		}	
	}

	// try to read the message meta from current chunk.
	// in event there not enough data, merge following chunk and try to read again.
	decodeNext() {
		if( this.cur_size === 0 || this.cur_message !== undefined ) return;

		let cur_chunk = this.buf[this.start];
		let cur_chunk_size = cur_chunk.byteLength - this.byte_offset;
		let msg_meta = this.decodeMeta(new DataView(cur_chunk.buffer, this.byte_offset));
		if( msg_meta !== undefined ) {
			this.byte_offset += msg_meta.meta_length;
			this.advanceChunk();
		}
		else if( this.cur_size > 1 ) {	// We didn't get enough data for a message, and there is another chunk available
			const next_i = this.nextReadI();
			const next_chunk = this.buf[next_i];
			const new_buf = new Uint8Array(cur_chunk_size + next_chunk.byteLength);
			new_buf.set(cur_chunk.slice(this.byte_offset));
			new_buf.set(next_chunk, cur_chunk_size);
			msg_meta = this.decodeMeta(new DataView(new_buf.buffer));
			if( msg_meta === undefined ) throw new Error("expected two chunks to be enough for a message meta");

			this.start = next_i;
			--this.cur_size;
			this.byte_offset = msg_meta.meta_length - cur_chunk_size;	// cur_chunk_size is the previous chunk. This only *looks* wrong.
		}

		if( msg_meta !== undefined ) {
			this.cur_message = msg_meta;
			this.pushChunkOnPayload();
		}
	}
	private decodeMeta(view:DataView) :incomingMessage|undefined {
		if( view.byteLength < 3 ) return;

		const msg :messageMeta = {
			service: view.getUint8(0),
			command: view.getUint8(1),
			msgID: view.getUint8(2),
			refMsgID: 0,
			payload: undefined
		}
		
		let offset = 3;

		if( msg.service === refRequestService ) {
			if( view.byteLength < 4 ) return;
			msg.refMsgID = view.getUint8(offset);
			++offset;
		}

		if( view.byteLength < offset +2 ) return;
		let pSize = view.getUint16(offset);
		offset += 2;
		if( pSize === 0xff ) {
			if( view.byteLength < offset +4 ) return;
			pSize = view.getUint32(offset);
			offset += 4;
		}

		return {msg, meta_length:offset, payload_remaining:pSize};
	}

	private pushChunkOnPayload() {
		if( this.cur_message === undefined ) throw new Error("no cur_message to push payload onto");
		
		if( this.cur_message.payload_remaining !== 0 ) {
			if( this.cur_message.msg.payload === undefined ) {
				this.cur_message.msg.payload = new Uint8Array(this.cur_message.payload_remaining);
			}
			const payload = this.cur_message.msg.payload;

			while(this.cur_message.payload_remaining && this.cur_size) {
				let payload_offset = payload.byteLength - this.cur_message.payload_remaining;

				let cur_chunk = this.buf[this.start];
				let read_length = Math.min(cur_chunk.byteLength - this.byte_offset, this.cur_message.payload_remaining);

				payload.set( cur_chunk.slice(this.byte_offset, this.byte_offset + read_length), payload_offset );
				this.cur_message.payload_remaining -= read_length;
				this.byte_offset += read_length;
				this.advanceChunk();
			}
		}
		
		if( this.cur_message.payload_remaining === 0 ) {
			if( this.resolveMessage !== undefined) {
				this.resolveMessage({value:this.cur_message.msg, done: false});
				this.resolveMessage = undefined;
				this.cur_message = undefined;
				this.decodeNext();
			}
		}
	}

	[Symbol.asyncIterator]() { return this; }
}