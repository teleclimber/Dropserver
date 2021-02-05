import {get} from '../controllers/userapi';
import {reactive} from 'vue';


// Email   string `json:"email"`
// UserID  int    `json:"user_id"`
// IsAdmin bool   `json:"is_admin"`

class User {
	loaded = false;

	user_id = -1;
	email = "";
	is_admin = false;

	async fetch() {
		const resp_data = await get('/user/');
		this.setFromRaw(resp_data);
	}
	setFromRaw(raw :any) {
		this.user_id = Number(raw.user_id);
		this.email = raw.email+"";
		this.is_admin = !!raw.is_admin;
		
		this.loaded = true;
	}
}

const user = reactive(new User);
user.fetch();

export default user;

export class AdminUsers {
	loaded = false;
	users :Map<number,User> = new Map;

	async fetch() {
		const resp_data = await get('/admin/user/');
		resp_data.forEach( (raw:any) => {
			const u = new User;
			u.setFromRaw(raw);
			this.users.set(u.user_id, u);
		});
		this.loaded = true;
	}
	get asArray() : User[] {
		return Array.from(this.users.values());
	}
}




