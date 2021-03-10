import WS from "jest-websocket-mock";
import { services, commands, encodeMessageMeta } from './twine-common';
import TwineWebsocketClient from './index'

test('websocket client connects', async () => {
	const addr = "ws://localhost:1234";
	const server = new WS(addr);

	server.nextMessage.then( m => {
		console.log(m);
		const msgMeta = encodeMessageMeta(2, 0, services.close, commands.ok, undefined);
		const send = new ArrayBuffer(5);
		new Uint8Array(send).set(msgMeta)
		server.send(send);
	});

	const t = new TwineWebsocketClient(addr);
	await t.startClient();

	WS.clean();
});
