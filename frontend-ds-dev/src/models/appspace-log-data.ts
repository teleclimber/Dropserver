import twineClient from './twine-client';
import {SentMessageI, ReceivedMessageI} from 'twine-web';

type AppspaceLogChunk = {
	from: number,
	to: number,
	content: string
}

export type AppspaceLogEntry = {
	appspace_id: number,
	time: Date,
	source: string,
	message: string
}

// outgoing commands:
const subscribeAppspaceLogCmd = 11
const subscribeAppLogCmd = 12

// incoming commands, ref to 11:
const statusSubCmd = 11
const chunkSubCmd = 12
const entrySubCmd = 13

// outgoing sub commands to 11
const unsubscribeLogCmd = 13

export default class LiveLog {
	subscribed = false;
	log_open  = false;
	entries :AppspaceLogEntry[] = [];

	_entries_ref_message :SentMessageI|undefined;

	async subscribeAppspaceLog(appspace_id :number) {
		if( this.subscribed ) throw new Error("This instance is already handing a log. Use new instance.");
		this.subscribed = true;
		await twineClient.ready();
		const payload = new TextEncoder().encode(JSON.stringify({appspace_id}))
		const sent = await twineClient.twine.send(15, subscribeAppspaceLogCmd, payload);
		this.handleMessages(sent)
	}
	async subscribeAppLog(app_id :number, version: string) {
		if( this.subscribed ) throw new Error("This instance is already handing a log. Use new instance.");
		this.subscribed = true;
		await twineClient.ready();
		const payload = new TextEncoder().encode(JSON.stringify({app_id, version}))
		const sent = await twineClient.twine.send(15, subscribeAppLogCmd, payload);
		this.handleMessages(sent)
	}
	async handleMessages(sent:SentMessageI) {
		for await (const m of sent.incomingMessages()) {
			switch (m.command) {
				case statusSubCmd:
					this.handleStatus(m);
					break;
				case chunkSubCmd:
					this.handleInitialChunk(m);
					break;
				case entrySubCmd:
					this.handleEntry(m);
					break;
				default:
					m.sendError("What is this command?");
					throw new Error("what is this command? "+m.command);
			}
		}
	}
	handleStatus(m :ReceivedMessageI) {
		if( m.payload === undefined ) {
			m.sendError("payload undefined");
			throw new Error("payload undefined");
		}
		m.sendOK();

		if( m.payload[0] == 0x00 ) {
			this.log_open = false;
			console.log("got log status closed");
		}
		else {
			this.log_open = true;
			console.log("got log status open");
			// have to reload at this point? Or can we let the backend do it?
		}
	}
	handleInitialChunk(m:ReceivedMessageI) {
		this.entries = [];	//reset entries when we get an initial chunk (it means log was just opened, so we don't know if old entries are still part of log)

		let chunk_data;
		try {
			chunk_data = <AppspaceLogChunk>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
		}
		catch(e) {
			m.sendError("error processing appspace log event "+e);
			console.error(e);
			return;
		}

		if( chunk_data.from == chunk_data.to || chunk_data.content == "" ) {
			// we got nothing.
			// we could try asking for more data, but it's likely there just isn't anything there.
			console.log("empty chunk");
			this.entries = [];
			m.sendOK();
			return;
		}

		try {
			this.entries = chunk_data.content.split("\n").filter( e => !!e ).map(parseEntry)
		}
		catch(e) {
			m.sendError("error processing appspace log event "+e);
			console.error(e);
			return;
		}

		m.sendOK();
	}
	handleEntry(m:ReceivedMessageI) {
		try {
			const entry = new TextDecoder('utf-8').decode(m.payload);
			this.entries.push(parseEntry(entry));
		}
		catch(e) {
			m.sendError("error processing appspace log event "+e);
			console.error(e);
			return;
		}

		m.sendOK();
	}
	async unsubscribeFromLog() {
		if( this._entries_ref_message === undefined ) return;
		const reply = await this._entries_ref_message.refSendBlock(unsubscribeLogCmd, undefined);
		if( reply.error ) {
			throw reply.error;
		}
		this._entries_ref_message = undefined;
	}
}

function parseEntry(entry :string) :AppspaceLogEntry {
	const parts = entry.split(" ");
	let message = parts.slice(2).join(" ").replaceAll('\\n', '\n').trim();

	const ret =  {
		appspace_id: 15,
		time: new Date(parts[0]),
		source: parts[1],
		message,
	}
	//console.log("parsed entry: ", entry, ret);
	return ret;
}
