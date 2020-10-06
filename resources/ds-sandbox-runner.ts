import * as path from "https://deno.land/std/path/mod.ts";

import DsServices from "./ds-services.ts";

async function run() {
	await DsServices.initTwine();
}

run();
