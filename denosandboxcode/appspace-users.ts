import DsServices from './ds-services.ts';

const service = 16;

const getUserCmd     = 12;
const getAllUsersCmd = 13;

export type User = {
	proxyId: string,
	permissions: string[],
	displayName: string
}

export default class Users {
	constructor(private services:DsServices) {}

	async get(proxyId: string) :Promise<User> {
		const twine = this.services.getTwine();
		const reply = await twine.sendBlock(service, getUserCmd, new TextEncoder().encode(proxyId));
		if(reply.error) {
			console.error("Failed to get user: "+reply.error);
			throw new Error(reply.error);
		}

		const user = <User> JSON.parse(new TextDecoder().decode(reply.payload));

		reply.sendOK();

		return user;
	}

	async getAll() :Promise<User[]> {
		const twine = this.services.getTwine();
		const reply = await twine.sendBlock(service, getAllUsersCmd, undefined);
		if(reply.error) {
			console.error("Failed to get users: "+reply.error);
			throw new Error(reply.error);
		}

		const users = <User[]> JSON.parse(new TextDecoder().decode(reply.payload));

		reply.sendOK();

		return users;
	}
}
