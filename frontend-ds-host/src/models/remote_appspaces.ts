import {get, post, del} from '../controllers/userapi';

export class RemoteAppspace {
	loaded = false;

	domain_name = "";
	owner_dropid = "";
	user_dropid = "";
	no_tls = false;
	port_string = "";
	created_dt = new Date();

	async fetch(domain_name: string) {
		const resp_data = await get('/remoteappspace/'+domain_name);
		this.setFromRaw(resp_data);
	}
	async refresh() {
		await this.fetch(this.domain_name);
	}
	setFromRaw(raw :any) {
		this.domain_name = raw.domain_name+'';
		this.owner_dropid = raw.owner_dropid+'';
		this.user_dropid = raw.user_dropid+'';
		this.no_tls = !!raw.no_tls;
		this.created_dt = new Date(raw.created_dt);

		this.loaded = true;
	}
	async del() {
		await del('/remoteappspace/'+this.domain_name)
	}
}

export class RemoteAppspaces {
	loaded = false;

	remotes : Map<string,RemoteAppspace> = new Map();

	async fetchForOwner() {
		const resp_data = await get('/remoteappspace');
		resp_data.forEach( (raw:any) => {
			const remote = new RemoteAppspace;
			remote.setFromRaw(raw);
			this.remotes.set(remote.domain_name, remote);
		});
		this.loaded = true;
	}

	get asArray() : RemoteAppspace[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		return Array.from(this.remotes.values());
	}
}

type RemoteAppspacePost = {
	check_only: boolean,
	domain_name: string,
	user_dropid: string,
}
export type RemoteAppspacePostResp = {
	inputs_valid: boolean,
	domain_message: string,
	remote_message: string
}

export function checkRemoteAppspace(domain_name:string, user_dropid: string) {
	const payload:RemoteAppspacePost = {
		check_only: true,
		domain_name,
		user_dropid
	} 

	const ret = post('/remoteappspace', payload)
	// it seems the response could contain all kinds of things.
	// Don't know if we'll have this yet, so leave unfinished.

}
export async function createRemoteAppspace(domain_name:string, user_dropid: string) :Promise<RemoteAppspacePostResp> {
	const payload:RemoteAppspacePost = {
		check_only: false,
		domain_name,
		user_dropid
	}

	const ret = await post('/remoteappspace', payload)
	// no return value? we expect caller to redirect to manage page.

	return <RemoteAppspacePostResp>ret;
}