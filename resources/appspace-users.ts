import DsServices from "./ds-services.ts";
import type Twine from "./twine/twine.ts";

const service = 16;

const getUserCmd     = 12;
const userIsOwnerCmd = 13;

type User = {
	proxy_id: string,
	permissions: string[],
	display_name: string
}

class Users {
	private twine: Twine;
	constructor() {
		this.twine = DsServices.getTwine();
	}

	async getUser(proxy_id: string) :Promise<User> {
		const reply = await this.twine.sendBlock(service, getUserCmd, new TextEncoder().encode(proxy_id));
		if(reply.error) {
			console.error("Failed to get user: "+reply.error);
			throw new Error(reply.error);
		}

		const user = <User> JSON.parse(new TextDecoder().decode(reply.payload));

		reply.sendOK();

		return user;
	}

	async isOwner(proxy_id: string) :Promise<boolean> {
		const reply = await this.twine.sendBlock(service, userIsOwnerCmd, new TextEncoder().encode(proxy_id));
		if(reply.error) {
			console.error("Failed to get is owner: "+reply.error);
			throw new Error(reply.error);
		}

		let is_owner = false;
		if( reply.command === 14 ) {
			is_owner = true;
		} else if( reply.command === 15 ) {
			is_owner = false;
		} else {
			throw new Error("Unexpected reply command from user service.")
		}

		reply.sendOK();
		
		return is_owner;
	}
}


const sym = Symbol.for("DropServer Users class singleton");
const w = <{[sym]?:Users}>window;
if(w[sym] === undefined) w[sym] = new Users;

export default w[sym] as Users;