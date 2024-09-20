import { ref, Ref, shallowRef, ShallowRef, triggerRef, shallowReactive, computed, version, isReactive } from 'vue';
import { defineStore } from 'pinia';
import { AppGetMeta, AppManifest } from './types';
import { on } from '../sse';
import { useAppsStore, rawToAppManifest } from './apps';
import { ax } from '../controllers/userapi';
import axios from 'axios';
import type {AxiosResponse, AxiosError} from 'axios';


// type InProcessResp struct {
//     LastEvent domain.AppGetEvent `json:"last_event"`
//     Meta      domain.AppGetMeta  `json:"meta"`
// }
// type AppGetEvent struct {
//     Key   AppGetKey `json:"key"`
//     Done  bool      `json:"done"`
//     Error bool      `json:"error"`
//     Step  string    `json:"step"`
// }
type AppGetEvent = {
	owner_id: number,
	key: string,
	done: boolean,
	input: string,
	step: string
}
type InProcessResp = {
	last_event: AppGetEvent,
	meta: AppGetMeta
}

export const useAppGetterStore = defineStore('app-getter', () => {

	const appgets : ShallowRef<Map<string,AppGetter>> = shallowRef(new Map());

	function loadKey(key:string) {
		// fetch latest event and data using normal get
		// if we have no data for the get, then apply..
		let appGet = get(key);
		if( !appGet ) {
			appGet = new AppGetter(key);
			appgets.value.set(key, appGet);
		}
		return appGet;
	}

	on('AppGetter', (raw) => {
		const appGetEv = <AppGetEvent>raw;
		const key = appGetEv.key;
		const appGet = get(key);
		if( !appGet ) appgets.value.set(key, new AppGetter(key));
		else appGet.updateEvent(appGetEv);
	});

	function get(key:string) {
		return appgets.value.get(key);
	}

	return { loadKey, get };
});


export class AppGetter {
	key:string;	// does this really need to be reactive? It should never chnage!

	not_found = ref(false);
	last_event :ShallowRef<AppGetEvent | undefined> = shallowRef();
	meta :ShallowRef<AppGetMeta | undefined> = shallowRef();

	constructor(key:string) {
		this.key = key;
		this.loadInProcess();
	}

	updateEvent(ev:AppGetEvent) {
		if( ev.done || ev.input ) this.loadInProcess();
		else this.last_event.value = ev;
	}

	async loadInProcess() {
		let resp :AxiosResponse|undefined;
		try {
			resp = await ax.get('/api/application/in-process/'+this.key);
		}
		catch(error: any | AxiosError) {
			if( axios.isAxiosError(error) && error.response && error.response.status == 404 ) {
				this.not_found.value = true;
				return;
			}
			throw error;
		}
		if( resp?.data === undefined ) return;
		const data = <InProcessResp>resp.data;
		if( data.meta.version_manifest ) data.meta.version_manifest = rawToAppManifest(data.meta.version_manifest);
		this.last_event.value = data.last_event;
		this.meta.value = data.meta;
		if( this.done && !this.has_error ) {
			const appStore = useAppsStore();
			appStore.setReload();	// temporary until we can lazy-load apps.
		}
	}

	get done() :boolean {	// Note we changed what "done" means on the backend, sort of. It now means the process is completely finished.
		return !!this.last_event.value?.done
	}
	get expects_input() :boolean {
		return !!this.last_event.value?.input;
	}
	get must_confirm() :boolean {
		return !this.done && this.last_event.value?.input === "commit";
	}
	get has_error() :boolean {
		return this.meta.value?.errors.length !== 0;
	}
	get version_manifest() :AppManifest|undefined {
		return this.meta.value?.version_manifest;
	}

	async cancel() {	// it might be this should be in app getter model?
		if( this.not_found.value ) return;
		try {
			this.not_found.value = true;
			this.last_event.value = undefined;
			this.meta.value = undefined;
			await ax.delete('/api/application/in-process/'+this.key);
		}
		catch(e) {
			// no-op. Whatever the error, don't hold up the frontend UI because of it.
		}
	}
}