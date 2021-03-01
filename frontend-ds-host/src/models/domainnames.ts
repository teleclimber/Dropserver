import {get, post} from '../controllers/userapi';

// temporary structure:
// type DomainData struct {
// 	DomainName string `json="domain_name"`
// }

export class DomainName {
	loaded = false;

	domain_name = '';

	setFromRaw(raw :any) {
		this.domain_name = raw.domain_name + '';

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
}