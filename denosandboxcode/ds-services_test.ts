import * as path from "https://deno.land/std@0.106.0/path/mod.ts";
import Twine from "./twine.ts";
import DsServices from './ds-services.ts';

Deno.test({
	name: "execute default function",
	//ignore: true,
	fn: async () => {

		const file = "testFile.ts";
		const dir = await Deno.makeTempDir();
		const code = "export default function testFn() {};";
		
		const full_path = path.join(dir, file);
		await Deno.writeFile(full_path, new TextEncoder().encode(code));

		const twine_sock = path.join(dir, "rev.sock");

		const twine_server = new Twine(twine_sock, true);
		const server_p = twine_server.startServer();

		const dsServices = new DsServices();
		dsServices.initTwine(twine_sock);

		await server_p;

		const send_data = {	file: full_path	};
		const reply = await twine_server.sendBlock(12, 11,  new TextEncoder().encode(JSON.stringify(send_data)));

		if( !reply.ok ) {
			console.error(reply.error);
			throw reply.error;
		}

		await twine_server.graceful();
	}
});
