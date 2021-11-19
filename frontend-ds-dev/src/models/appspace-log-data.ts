import {reactive} from 'vue';
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

// AppspaceLogData 
class AppspaceLogData {
	log_open  = false;
	entries :AppspaceLogEntry[] = [];

	_entries_ref_message :SentMessageI|undefined;
	_start() {
		this.getLogs();
	}
	async getLogs() {
		await twineClient.ready();
		const payload = new TextEncoder().encode(JSON.stringify({appspace_id:15}))
		const sent = await twineClient.twine.send(15, 11, payload);
		for await (const m of sent.incomingMessages()) {
			switch (m.command) {
				case 11:
					this.handleStatus(m);
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
			this.unsubscribeFromTailLogs();
		}
		else {
			this.log_open = true;
			console.log("got log status open")
			this.tailLogs();
		}
	}
	async tailLogs() {
		const payload = new TextEncoder().encode(JSON.stringify({appspace_id:15}))
		this._entries_ref_message = await twineClient.twine.send(15, 12, payload);
		for await (const m of this._entries_ref_message.incomingMessages()) {
			switch (m.command) {
				case 11:
					this.handleLogChunk(m);
					break;
				case 12:
					this.handleLogEntry(m);
					break;
				default:
					m.sendError("What is this command?");
					throw new Error("what is this command? "+m.command);
			}
		}
	}
	handleLogChunk(m:ReceivedMessageI) {
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
	handleLogEntry(m:ReceivedMessageI) {
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
	async unsubscribeFromTailLogs() {
		if( this._entries_ref_message === undefined ) return;
		const reply = await this._entries_ref_message.refSendBlock(13, undefined);
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

const appspaceLogData = reactive(new AppspaceLogData());
appspaceLogData._start();

export default appspaceLogData;