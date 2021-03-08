import {get, post} from '../controllers/userapi';

// temporary structure:
// type DomainData struct {
// 	DomainName                string `json:"domain_name"`
// 	UserOwned                 bool   `json:"user_owned"`
// 	ForAppspace               bool   `json:"for_appspace"`
// 	AppspaceSubdomainRequired bool   `json:"appspace_subdomain_required"`
// 	ForDropID                 bool   `json:"for_dropid"`
// 	DropIDSubdomainAllowed    bool   `json:"dropid_subdomain_allowed`
// }

export class DomainName {
	loaded = false;

	domain_name = '';
	user_owned = false;
	for_appspace = false;
	appspace_subdomain_required = false;
	for_dropid = false;
	dropid_subdomain_allowed = false;

	setFromRaw(raw :any) {
		this.domain_name = raw.domain_name + '';
		this.user_owned = !!raw.user_owned;
		this.for_appspace = !!raw.for_appspace;
		this.appspace_subdomain_required = !!raw.appspace_subdomain_required;
		this.for_dropid = !!raw.for_dropid;
		this.dropid_subdomain_allowed = !!raw.dropid_subdomain_allowed;

		this.loaded = true;
	}



}

export class DomainNames {
	loaded = false;

	domains : DomainName[] = [];

	async fetchForOwner() {
		const resp_data = await get('/domainname');
		resp_data.forEach( (raw:any) => {
			const dn = new DomainName;
			dn.setFromRaw(raw);
			this.domains.push(dn);
		});
		this.loaded = true;
	}

	get for_dropid() : DomainName[] {
		return this.domains.filter((d) => d.for_dropid);
	}

	get for_appspace() : DomainName[] {
		//console.log("calculating for appspace "+this.domains.length);
		return this.domains.filter((d) => d.for_appspace);
	}
}

type CheckDomainResult = {
	valid: boolean,
	available: boolean,
	message:string
}

export async function checkAppspaceDomain(domain_name:string, subdomain:string) :Promise<CheckDomainResult> {
	let q = 'appspace'
			+'&domain='+encodeURIComponent(domain_name)
			+'&subdomain='+encodeURIComponent(subdomain);
	return <CheckDomainResult>await	get('/domainname/check?'+q);
}