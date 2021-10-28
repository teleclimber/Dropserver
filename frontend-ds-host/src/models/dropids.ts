import {get, post} from '../controllers/userapi';

// type DropID struct {
// 	UserID      UserID    `db:"user_id" json:"user_id" validate:"nonzero"`
// 	Handle      string    `db:"handle" json:"handle" validate:"nonzero,max=100"`
// 	Domain      string    `db:"domain" json:"domain" validate:"nonzero,max=100"`
// 	DisplayName string    `db:"display_name" json:"display_name" validate:"max=100"`
// 	Created     time.Time `db:"created" json:"created_dt"`
// }

export class DropID {
	loaded = false;

	user_id = 0;
	handle = "";
	domain_name = "";
	display_name = "";
	created_dt = new Date();

	key = "";	// key is used to identify the dropid uniquely
	// this should actually be called "full", and should be a computed getter

	setFromRaw(raw :any) {
		this.user_id = Number(raw.user_id);
		this.handle = raw.handle + '';
		this.domain_name = raw.domain + '';
		this.display_name = raw.display_name + '';
		this.created_dt = new Date(raw.created_dt);

		this.key = this.domain_name+"/"+this.handle;

		this.loaded = true;
	}

	get full() {
		let f = this.domain_name;
		if( this.handle !== "" ) f = this.domain_name+'/'+this.handle;
		return f;
	}

}

export class DropIDs {
	loaded = false;
	dropids :DropID[] = [];
	
	async fetchForOwner() {
		const resp_data = await get('/dropid');
		resp_data.forEach( (raw:any) => {
			const dropid = new DropID;
			dropid.setFromRaw(raw);
			this.dropids.push(dropid);
		});
		this.loaded = true;
	}

}

export async function createDropID(handle:string, domain:string, display_name: string) :Promise<DropID> {
	const resp_data = await post('/dropid', {handle, domain, display_name});
	const dropid = new DropID;
	dropid.setFromRaw(resp_data);
	return dropid;
}

export async function checkHandle(handle:string, domain:string) :Promise<boolean> {
	let q = 'domain='+encodeURIComponent(domain);
	if( handle !== '' ) q += '&handle='+encodeURIComponent(handle)
	try {
		await get('/dropid?'+q);
	} catch(error:any) {
		if( error.response.status == 404 ) return true;
		return false;
	}
	return false;	// request return 200, meaning it was found, meaning it's unavailable
}