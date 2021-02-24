import {get, post} from '../controllers/userapi';

// type ContactResp struct {
// 	UserID      domain.UserID    `json:"user_id"`
// 	ContactID   domain.ContactID `json:"contact_id"`
// 	Name        string           `json:"name"`
// 	DisplayName string           `json:"display_name"`
// 	Created     time.Time        `json:"created_dt"`
// 	// proabbly include array of authentication methods
// }

export class Contact {
	loaded = false;

	user_id = -1;
	contact_id = -1;
	name = '';
	display_name = '';
	created_dt = new Date();

	// ...

	async fetch(contact_id: number) {
		const resp_data = await get('/contact/'+contact_id);
		this.setFromRaw(resp_data);
	}
	setFromRaw(raw :any) {
		this.user_id = Number(raw.user_id);
		this.contact_id = Number(raw.contact_id);
		this.name = raw.name + '';
		this.display_name = raw.display_name + '';
		this.created_dt = new Date(raw.created_dt);

		this.loaded = true;
	}

}

export class Contacts {
	loaded = false;
	contacts :Map<number,Contact> = new Map();
	
	async fetchForOwner() {
		const resp_data = await get('/contact');
		resp_data.forEach( (raw:any) => {
			const contact = new Contact;
			contact.setFromRaw(raw);
			this.contacts.set(contact.contact_id, contact);
		});
		this.loaded = true;
	}

	get asArray() : Contact[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		return Array.from(this.contacts.values());
	}
}

export async function createContact(name: string, display_name: string) :Promise<Contact> {
	const resp_data = await post('/contact', {name, display_name});
	const contact = new Contact;
	contact.setFromRaw(resp_data);
	return contact;
}