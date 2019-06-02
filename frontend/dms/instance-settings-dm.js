import axios from 'axios';

import { action, computed, observable, decorate, runInAction, flow } from "mobx";

class InstanceSettingsDM {
	constructor() {
		//this.loaded = false;
	}

	async fetchData() {
		const resp = await axios.get( '/api/admin/settings' );
		runInAction( () => { this.data = resp.data;	} );
	}

	async commitData( data ) {
		const resp = await axios.patch( '/api/admin/settings', data );
		runInAction( () => { this.data = resp.data;	} );
	}

	
}

decorate( InstanceSettingsDM, {
	//loaded: observable,
	data: observable
});

export default InstanceSettingsDM;