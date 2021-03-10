import {get, post} from '../controllers/userapi';

//type UserInvitation struct {
// 	Email string `db:"email" json:"email"`
// }

export class Invitation {
	loaded = false;
	email = "";

	setFromRaw(raw :any) {
		this.email = raw.email+"";
		this.loaded = true;
	}
}

export class AdminInvitations {
	loaded = false;
	invitations :Map<string,Invitation> = new Map;

	async fetch() {
		const resp_data = await get('/admin/invitation/');
		if( resp_data ) {
			resp_data.forEach( (raw:any) => {
				const i = new Invitation;
				i.setFromRaw(raw);
				this.invitations.set(i.email, i);
			});
		}
		this.loaded = true;
	}
	get asArray() : Invitation[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		return Array.from(this.invitations.values());
	}

	async createInvitation(email:string) {
		await post('/admin/invitation', {email});
		const inv = new Invitation;
		inv.email = email;
		this.invitations.set(email, inv);
	}
}