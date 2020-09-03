import {reactive} from 'vue';

const baseData = reactive({
	app_path: "/",
	app_name: "some app name",
	app_version: "0.0.0",
	app_schema: 0,

	appspace_path: "/",
	appspace_schema: 0,
});

async function fetchData() {
	try {
		const res = await fetch('base-data');
		if( !res.ok ) {
			throw new Error("fetch error for basic data");
		}
		const data = await res.json();
		Object.assign(baseData, data);
	}
	catch(error) {
		console.error(error);
	}
}

export {
	baseData,
	fetchData,
};

