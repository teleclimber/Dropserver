import ds_axios from '../ds-axios-helper.js'

import { action, computed, observable, decorate, runInAction, flow } from "mobx";

class InstanceSettingsDM {
	constructor() {
	}

	async fetchData() {
		ds_axios.get( '/api/admin/settings' ).then( resp => {
			runInAction( () => {
				let settings = resp.data.settings;
				settings.registration_open = settings.registration_open ? 'open' : 'closed';
				this.data = settings;
			});
		}).catch( e => {
			console.error(e);
		});
	}

	async commitData( commit_data ) {
		const patch_data = {
			registration_open: commit_data.registration_open == 'open' ? true : false
		}
		ds_axios.patch( '/api/admin/settings', {settings: patch_data} ).catch( e => console.error(e) );
		runInAction( () => this.data = commit_data );
	}
}

decorate( InstanceSettingsDM, {
	data: observable
});

export default InstanceSettingsDM;