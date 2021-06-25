import * as path from "https://deno.land/std@0.97.0/path/mod.ts";
import Metadata from './ds-metadata.ts';
import AppRouter from './app-router.ts';
import type {RouteExport} from './app-router.ts';
import type {ReceivedMessageI} from "./twine/twine.ts";

const get_app_routes_cmd = 11;

export async function handleMessage(message :ReceivedMessageI) {
	switch (message.command) {
		case get_app_routes_cmd:
			await getAppRoutes(message);
			break;
	
		default:
			await message.sendError("Command not recognized");
	}
}

async function getAppRoutes(message :ReceivedMessageI) {
	let mod:any;
	try {
		mod = await import(path.join(Metadata.app_path, "router.ts"));	// def don't hardcode router.ts!
	}
	catch(e) {
		await message.sendError(e.toString());
		return;
	}

	// here we asusme the router module exported an AppRouter with all the desired routes.
	const app_router = <AppRouter>mod.default;

	let routes :RouteExport[];
	try {
		routes = app_router.exportStack();
	}
	catch(e) {
		await message.sendError(e.toString());
		return;
	}

	message.reply(11, new TextEncoder().encode(JSON.stringify(routes)));
}