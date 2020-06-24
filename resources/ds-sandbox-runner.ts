import * as path from "https://deno.land/std/path/mod.ts";
import DsServices from "./ds-services.ts";

let sock_path = Deno.args[Deno.args.length -3];
let app_path = Deno.args[Deno.args.length -2];	// is it really necessary to pass these in?
let appspace_path = Deno.args[Deno.args.length -1];

const rev_sock_path = path.join(sock_path, "rev.sock");

console.log("ds-sandbox-runner is running");

async function run() {
	await DsServices.initTwine(rev_sock_path);
}

run();
