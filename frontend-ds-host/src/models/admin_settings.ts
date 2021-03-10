import {get, patch} from '../controllers/userapi';

// Settings represents admin-settable parameters
// type Settings struct {
// 	RegistrationOpen bool `json:"registration_open" db:"registration_open"`
// }

type SettingsPayload = {
	registration_open: boolean
}

export class AdminSettings {
	loaded = false;
	registration_open = false;

	setFromRaw(raw :any) {
		this.registration_open = !!raw.registration_open;
		this.loaded = true;
	}
	async fetch() {
		const resp_data = await get('/admin/settings');
		this.setFromRaw(resp_data);
	}

	async setRegistrationOpen(open:boolean) {
		const payload = this.createSettingsPayload();
		payload.registration_open = open;
		await patch('/admin/settings', payload);
		this.registration_open = open;
	}

	createSettingsPayload() :SettingsPayload{
		return {
			registration_open: this.registration_open
		};
	}
}