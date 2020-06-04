import * as path from "https://deno.land/std/path/mod.ts";

let sock_path = Deno.args[Deno.args.length -3];
let app_path = Deno.args[Deno.args.length -2];
let appspace_path = Deno.args[Deno.args.length -1];

sock_path = path.join(sock_path, "rev.sock");

// const commandCodes = {
// 	""
// }

// command codes:
// - 1: Hi
// - 2: Bye
// - 3: Log (general log)
// - 4: log in appspace (like it's a console.log, although isn't that coming through std out?)
// - 10: db exec
// - 11: db select	// are these even separate codes?

// does it really need to be characters? It seems almost counter-productive.
// Just use integers <256



console.log("sock path: ", sock_path);



// Q: how do we match responses with requests? 

// function toBytesInt32 (num:number) {
// 	if( num > 255 ) throw new Error("num bigger than 255");

//     return new Uint8Array([
//          (num & 0xff000000) >> 24,
//          (num & 0x00ff0000) >> 16,
//          (num & 0x0000ff00) >> 8,
//          (num & 0x000000ff)
//     ]);
//     return arr.buffer;
// }



class HostConn {
	conn: Deno.Conn | undefined

	constructor(private sock_path: string) {

	}
	async connect() {
		this.conn = await Deno.connect({ address: this.sock_path, transport: "unix" });

		await this.sendCommand(1);	// "hi"
	}
	async disconnect() {
		if( !this.conn ) return;

		await this.sendCommand(2);	// "bye"
	}
	async sendCommand(cmd: number){
		if( !this.conn ) throw new Error("can't send: conn undefined");
		if( cmd > 255 ) throw new Error("command invalid: greater than 255");

		let uint8 = new Uint8Array(1);
		uint8[0] = cmd;
		let written = await this.conn.write(uint8);

	}
}

async function run() {
	const twine = new Twine(sock_path);
	await twine.startClient();

	//setTimeout( hostConn.disconnect, 100 );
}



run();

