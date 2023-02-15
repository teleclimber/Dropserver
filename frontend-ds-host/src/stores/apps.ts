import { ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import {get, post, del} from '../controllers/userapi';

enum LoadState {
	NotLoaded = 0,
	Loading = 1,
	Loaded = 2
}

interface AppVersion {
	app_id: number,
	version: string,
	app_name: string,	// unused?
	api_version: number,	
	schema: number,
	created_dt: Date
}

interface App {
	app_id: number,
	name: string,
	created_dt: Date,
	versions: AppVersion[]
}

function appVersionFromRaw(raw:any) {
	return {
		app_id: Number(raw.app_id),
		version: raw.version+'',
		app_name: raw.app_name+'',
		api_version: Number(raw.api_version),
		schema: Number(raw.schema),
		created_dt: new Date(raw.created_dt)
	}
}
function appFromRaw(raw:any) :App {
	const app_id = Number(raw.app_id);
	const name = raw.name + '';
	const created_dt = new Date(raw.created_dt);

	let versions = [];
	if( Array.isArray(raw.versions) ) versions = raw.versions.reverse().map(appVersionFromRaw);
	
	return {
		app_id,
		name,
		created_dt,
		versions
	}
}

export const useAppsStore = defineStore('apps', () => {
	const load_state = ref(LoadState.NotLoaded);

	const apps : ShallowRef<Map<number,ShallowRef<App>>> = shallowRef(new Map());

	const isLoaded = computed( () => {
		return load_state.value === LoadState.Loaded;
	});

	async function fetchForOwner() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp_data = await get('/application');
			resp_data.apps.forEach( (raw:any) => {
				const app = appFromRaw(raw);
				apps.value.set(app.app_id, shallowRef(app));
			});
			load_state.value = LoadState.Loaded;
		}
	}

	return {isLoaded, fetchForOwner, apps}
});