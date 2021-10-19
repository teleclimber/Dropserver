import DsServices from "./ds-services.ts";
import type Twine from "./twine.ts";

const service = 16;

const getUserCmd     = 12;
const getAllUsersCmd = 13;

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

	async getAllUsers() :Promise<User[]> {
		const reply = await this.twine.sendBlock(service, getAllUsersCmd, undefined);
		if(reply.error) {
			console.error("Failed to get users: "+reply.error);
			throw new Error(reply.error);
		}

		const users = <User[]> JSON.parse(new TextDecoder().decode(reply.payload));

		reply.sendOK();

		return users;
	}
}


const sym = Symbol.for("DropServer Users class singleton");
const w = <{[sym]?:Users}>window;
if(w[sym] === undefined) w[sym] = new Users;

export default w[sym] as Users;