import {get, post, patch} from '../controllers/userapi';

// type AppspaceUser struct {
// 	AppspaceID  AppspaceID         `db:"appspace_id" json:"appspace_id"`
// 	ProxyID     ProxyID            `db:"proxy_id" json:"proxy_id"`
// 	AuthType    string             `db:"auth_type" json:"auth_type"`
// 	AuthID      string             `db:"auth_id" json:"auth_id"`
// 	DisplayName string             `db:"display_name" json:"display_name"`
// 	Permissions string             `db:"permissions" json:"permissions"` // this is unfortunate because we'd like to split them first
// 	Created     time.Time          `db:"created" json:"created_dt"`
// 	LastSeen    nulltypes.NullTime `db:"last_seen" json:"last_seen"`
// }

export class AppspaceUser {
	loaded = false;

	appspace_id = 0;
	proxy_id = "";
	auth_type = "";
	auth_id = "";
	display_name = "";
	permissions :string[] = [];
	created_dt = new Date();
	last_seen :Date|undefined;

	setFromRaw(raw :any) {
		this.appspace_id = Number(raw.appspace_id);
		this.proxy_id = raw.proxy_id+'';
		this.auth_type = raw.auth_type+'';
		this.auth_id = raw.auth_id+'';
		this.display_name = raw.display_name+'';
		this.permissions = raw.permissions === "" ? [] : raw.permissions.split(",");	//are we doing commas. Ideally we'd get this as an array from above
		this.created_dt = new Date(raw.created_dt);
		this.last_seen = raw.last_seen ? new Date(raw.last_seen) : undefined;

		this.loaded = true;
	}
	async fetch(appspace_id: number, proxy_id:string) {
		const resp_data = await get('/appspace/'+appspace_id+'/user/'+proxy_id);
		this.setFromRaw(resp_data);
	}
}

export class AppspaceUsers {
	loaded = false;

	au :AppspaceUser[] = [];

	async fetchForAppspace(appspace_id:number) {
		const resp_data = await get('/appspace/'+appspace_id+'/user');
		resp_data.forEach( (raw:any) => {
			const appspace_user = new AppspaceUser;
			appspace_user.setFromRaw(raw);
			this.au.push(appspace_user);
		});
		this.loaded = true;
	}

}

export type AppspaceUserAuth = {
	auth_type: 'email'|'dropid',
	auth_id: string
}
export type AppspaceUserMeta = {
	display_name: string,
	permissions: string[]
}
export type PostAppspaceUser = {
	auth_type: string,
	auth_id: string,
	display_name: string,
	permissions: string[]
}
export async function saveNewUser(appspace_id:number, data :PostAppspaceUser ) {
	const resp_data = await post('/appspace/'+appspace_id+'/user', data);
}

export async function updateUserMeta(appspace_id:number, proxy_id:string, data:AppspaceUserMeta) {
	const resp_data = await patch('/appspace/'+appspace_id+'/user/'+proxy_id, data);
}