import * as path from "https://deno.land/std/path/mod.ts";
import Twine from "./twine/twine.ts";
import Metadata from "./ds-metadata.ts";

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

		const orig_rev_sock_path = Metadata.rev_sock_path;
		Metadata.rev_sock_path = twine_sock;

		const twine_server = new Twine(twine_sock, true);
		const server_p = twine_server.startServer();

		const services_module = await import("./ds-services.ts");	// import after stubbing
		const DsServices = services_module.default;

		//@ts-ignore
		DsServices.twine = undefined;	// have to reset because it errors if you try to init twine twice.
		DsServices.initTwine();

		await server_p;

		const send_data = {	file: full_path	};
		const reply = await twine_server.sendBlock(12, 11,  new TextEncoder().encode(JSON.stringify(send_data)));

		if( !reply.ok ) {
			console.error(reply.error);
			throw reply.error;
		}

		await twine_server.graceful();

		Metadata.rev_sock_path = orig_rev_sock_path;
	}
});
