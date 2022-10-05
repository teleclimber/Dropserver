import * as path from "https://deno.land/std@0.158.0/path/mod.ts";

export class Metadata {
	sock_path:string;
	app_path:string;
	appspace_path:string;
	avatars_path:string;
	rev_sock_path:string;

	constructor() {
		this.sock_path = arg2string(3);// Deno.args[Deno.args.length -3];
		this.app_path = arg2string(2);// Deno.args[Deno.args.length -2];
		this.appspace_path = path.join(arg2string(1), "files");// Deno.args[Deno.args.length -1];
		this.avatars_path = path.join(arg2string(1), "avatars");
		this.rev_sock_path = path.join(this.sock_path, "rev.sock");
	}
}

function arg2string(i:number) :string{
	let s = Deno.args[Deno.args.length -i];
	if( !s ) s = "";
	return s;
}

export default Metadata;