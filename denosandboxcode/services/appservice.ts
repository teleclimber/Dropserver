import AppRoutes from '../approutes.ts';
import type {RouteExport} from '../approutes.ts';
import type {ReceivedMessageI} from "./twine.ts";

const get_app_routes_cmd = 11;

export default class DsAppService {

	constructor(private appRoutes:AppRoutes) {}

	async handleMessage(message :ReceivedMessageI) {
		switch (message.command) {
			case get_app_routes_cmd:
				await this.getAppRoutes(message);
				break;
		
			default:
				await message.sendError("Command not recognized");
		}
	}

	async getAppRoutes(message :ReceivedMessageI) {
		let routes :RouteExport[];
		try {
			// load them first, which should amount to no-op if already loaded
			routes = this.appRoutes.exportStack();
		}
		catch(e) {
			console.error('Error getting routes: '+e);
			await message.sendError(e.toString());
			return;
		}

		message.reply(11, new TextEncoder().encode(JSON.stringify(routes)));
	}
}