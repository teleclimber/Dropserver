import * as path from "https://deno.land/std/path/mod.ts";

//import Metadata from "./ds-metadata.ts";
import DsServices from "./ds-services.ts";


console.log("ds-sandbox-runner is running");

async function run() {
	await DsServices.initTwine();
}

run();
