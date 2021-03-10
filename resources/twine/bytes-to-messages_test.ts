import Twine, {BytesToMessages} from "./twine.ts";
import { assertEquals } from "https://deno.land/std/testing/asserts.ts";

Deno.test({
	name: "send incomplete chunk.",
	//ignore: true,
	fn: async () => {
		const b2m = new BytesToMessages;
		b2m.push(new Uint8Array(2));
	}
});

Deno.test({
	name: "send two messages in two finite chunks.",
	//ignore: true,
	fn: async () => {
		const b2m = new BytesToMessages;

		// @ts-ignore
		let m = Twine.encodeMessageMeta(2, 0, 11, 7, undefined);
		b2m.push(m);

		let m_out = await b2m.next();
		let message = m_out.value;
		if( message === undefined ) throw new Error("message should not be undefined");
		assertEquals(message.msgID, 2);
		assertEquals(message.refMsgID, 0);
		assertEquals(message.service, 11);
		assertEquals(message.command, 7);
		assertEquals(message.payload, undefined);

		// @ts-ignore
		m = Twine.encodeMessageMeta(2, 0, 11, 77, undefined);
		b2m.push(m);

		m_out = await b2m.next();
		assertEquals(m_out.value?.command, 77);
	}
});

Deno.test({
	name: "send two messages in single chunk.",
	//ignore: true,
	fn: async () => {
		const b2m = new BytesToMessages;

		// @ts-ignore
		let m1 = Twine.encodeMessageMeta(2, 0, 11, 7, undefined);
		// @ts-ignore
		let m2 = Twine.encodeMessageMeta(3, 0, 12, 8, undefined);
		const chunk = new Uint8Array(m1.byteLength + m2.byteLength);
		chunk.set(m1, 0);
		chunk.set(m2, m1.byteLength);

		b2m.push(chunk);
		
		let m_out = await b2m.next();
		assertEquals(m_out.value?.command, 7);

		m_out = await b2m.next();
		assertEquals(m_out.value?.command, 8);
	}
});

Deno.test({
	name: "message meta split into 2 chunks.",
	//ignore: true,
	fn: async () => {
		const b2m = new BytesToMessages;

		// @ts-ignore
		let m1 = Twine.encodeMessageMeta(2, 0, 11, 7, undefined);
		// @ts-ignore
		let m2 = Twine.encodeMessageMeta(3, 0, 12, 8, undefined);
		const chunk = new Uint8Array(m1.byteLength + m2.byteLength);
		chunk.set(m1, 0);
		chunk.set(m2, m1.byteLength);

		b2m.push(chunk.slice(0,2));
		b2m.push(chunk.slice(2));
		
		let m_out = await b2m.next();
		assertEquals(m_out.value?.command, 7);

		m_out = await b2m.next();
		assertEquals(m_out.value?.command, 8);
	}
});

Deno.test({
	name: "message meta split and get next promise.",
	//ignore: true,
	fn: async () => {
		const b2m = new BytesToMessages;

		// @ts-ignore
		let m1 = Twine.encodeMessageMeta(2, 0, 11, 7, undefined);
		// @ts-ignore
		let m2 = Twine.encodeMessageMeta(3, 0, 12, 8, undefined);
		const chunk = new Uint8Array(m1.byteLength + m2.byteLength);
		chunk.set(m1, 0);
		chunk.set(m2, m1.byteLength);

		b2m.push(chunk.slice(0,2));

		const m_out_p = b2m.next();
		b2m.push(chunk.slice(2));
		
		let m_out = await m_out_p;
		assertEquals(m_out.value?.command, 7);

		m_out = await b2m.next();
		assertEquals(m_out.value?.command, 8);
	}
});

Deno.test({
	name: "message payload split into 2 chunks.",
	//ignore: true,
	fn: async () => {
		const b2m = new BytesToMessages;

		const payload_str = "hello world";
		const payload = new TextEncoder().encode(payload_str);

		// @ts-ignore
		let m1 = Twine.encodeMessageMeta(2, 0, 11, 7, payload);
		// @ts-ignore
		let m2 = Twine.encodeMessageMeta(3, 0, 12, 8, undefined);
		const chunk = new Uint8Array(m1.byteLength + payload.byteLength + m2.byteLength);
		chunk.set(m1, 0);
		chunk.set(payload, m1.byteLength);
		chunk.set(m2, payload.byteLength + m1.byteLength);

		b2m.push(chunk.slice(0,10));
		let m_out_p = b2m.next();

		b2m.push(chunk.slice(10));
		
		let m_out = await m_out_p;
		assertEquals(m_out.value?.command, 7);
		assertEquals(m_out.value?.payload, payload);

		m_out = await b2m.next();
		assertEquals(m_out.value?.command, 8);
	}
});


