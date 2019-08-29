import ds_axios from '../ds-axios-helper.js'

import { action, computed, observable, decorate, runInAction, flow } from "mobx";

class InstanceSettingsDM {
	constructor() {
		//this.loaded = false;
	}

	async fetchData() {
		ds_axios.get( '/api/admin/settings' ).then( resp => {
			runInAction( () => { this.data = resp.data;	} );
		}).catch( e => {
			console.error(e);
		});
	}

	async commitData( data ) {
		const resp = await ds_axios.patch( '/api/admin/settings', data );
		runInAction( () => { this.data = resp.data;	} );
	}

	
}

decorate( InstanceSettingsDM, {
	//loaded: observable,
	data: observable
});

export default InstanceSettingsDM;