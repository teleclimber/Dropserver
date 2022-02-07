import {reactive} from 'vue';

class BaseData {
	loaded = false;

	app_path = "";
	appspace_path = "";
	appspace_working_dir = "";

	_start() {
		this.fetchInitialData();
	}
	async fetchInitialData() {
		const res = await fetch('base-data');
		if( !res.ok ) {
			throw new Error("fetch error for basic data");
		}

		try {
			const data = await res.json();
			Object.assign(this, data);
		}
		catch(error) {
			console.error(error);
		}
		this.loaded = true;
	}
}

const baseData = reactive(new BaseData());
baseData._start();
export default baseData;


