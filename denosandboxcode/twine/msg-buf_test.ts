import Twine, {MessageBuffer, ReceivedMessageI, Message} from "./twine.ts";
import * as path from "https://deno.land/std@0.106.0/path/mod.ts";
import { assertEquals } from "https://deno.land/std@0.106.0/testing/asserts.ts";

Deno.test("buf use buf", async () => {
	const mb = new MessageBuffer();

	const m = makeMessage(12);

	mb.push(m);

	const in_m = await mb.next();
	if(in_m.value === undefined) throw new Error("value should not be undefined");
	assertEquals(m.msgID, in_m.value.msgID);
});

Deno.test("buf use promise", async () => {
	const mb = new MessageBuffer();

	const m = makeMessage(12);

	setTimeout( () => {
		mb.push(m);
	}, 0);

	const in_m = await mb.next();
	if(in_m.value === undefined) throw new Error("value should not be undefined");
	assertEquals(m.msgID, in_m.value.msgID);
});

Deno.test("buf", async () => {
	const mb = new MessageBuffer();

	let id = 10;
	for( let i=0; i<3; ++i ) {
		mb.push(makeMessage(id));
		++id;
	}

	const wg = new WaitGroup(6);
	wg.waitDone().then(() => mb.stop());

	setTimeout( () => {
		for( let i=0; i<3; ++i ) {
			mb.push(makeMessage(id));
			++id;
		}
	}, 0);
	
	let out_id = 10;
	for await (const m of mb ) {
		if(m === undefined) throw new Error("should not be undefined");
		assertEquals(out_id, m.msgID);
		++out_id;
		wg.done();
	}

	assertEquals(16, id);
	assertEquals(16, out_id);
});

function makeMessage(msgID: number) :ReceivedMessageI {
	return new Message( {
		msgID: msgID,
		refMsgID: 0,
		service: 7,
		command: 11,
		payload: undefined
	}, new Twine("", false));

}

class WaitGroup {
	resolve: (()=>void)|undefined;
	constructor(private count:number) {}
	done() {
		--this.count;
		if( this.count === 0 && this.resolve !== undefined ) {
			this.resolve();
		}
	}
	async waitDone():Promise<void> {
		return new Promise( (resolve) => {
			if( this.resolve !== undefined) throw new Error("already witing on done");
			if( this.count === 0 ) resolve();
			else this.resolve = resolve;
		});
	}
}