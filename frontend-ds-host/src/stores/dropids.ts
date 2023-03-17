import { ref, computed, ShallowRef, shallowRef } from 'vue';
import { defineStore } from 'pinia';
import {ax} from '@/controllers/userapi';
import { LoadState, UserDropID } from './types';

export const useDropIDsStore = defineStore('user-dropids', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const dropids : ShallowRef<Map<string,ShallowRef<UserDropID>>> = shallowRef(new Map);
	
	async function loadData() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp = await ax.get('/api/dropid');
			if( !Array.isArray(resp.data) ) throw new Error("expected array for dropds, got "+typeof resp.data);
			const dids :Map<string,ShallowRef<UserDropID>> = new Map; 
			resp.data.forEach( (raw) => {
				const d = dropidFromRaw(raw);
				dids.set(d.compound_id, shallowRef(d));
			});
			dropids.value = dids;
			load_state.value = LoadState.Loaded;
		}
	}

	async function createDropID(handle:string, domain:string, display_name: string) :Promise<UserDropID> {
		const resp_data = await ax.post('/api/dropid', {handle, domain, display_name});
		const dropid = dropidFromRaw(resp_data.data);
		dropids.value.set(dropid.compound_id, shallowRef(dropid));
		dropids.value = new Map(dropids.value);
		return dropid;
	}

	async function updateDropID(handle:string, domain:string, display_name: string) :Promise<void> {
		const dropid = dropids.value.get(getCompoundID(domain, handle));
		if( dropid === undefined ) throw new Error("dropid not found, can not update.")
		try {
			await ax.patch('/api/dropid?'+getQueryString(handle, domain), {display_name});
		}
		catch(e) {
			throw new Error("error updating display_name: "+e);
		}
		dropid.value = Object.assign({}, dropid.value, {display_name})
	}
	
	async function checkHandle(handle:string, domain:string) :Promise<boolean> {
		const resp = await ax.get('/api/dropid?check=yes&'+getQueryString(handle, domain));
		return !!resp.data.available;
	}

	function getDropID(domain:string, handle:string) :ShallowRef<UserDropID>|undefined {
		return dropids.value.get(getCompoundID(domain, handle));
	}

	return {loadData, is_loaded, dropids, createDropID, updateDropID, checkHandle, getDropID};
})

function dropidFromRaw(raw:any) :UserDropID {
	const handle = raw.handle + '';
	const domain_name = raw.domain + '';
	return {
		user_id: Number(raw.user_id),
		handle,
		domain_name,
		compound_id: getCompoundID(domain_name, handle),
		display_name: raw.display_name + '',
		created_dt: new Date(raw.created_dt)
	};
}

function getCompoundID(domain :string, handle: string) :string {
	if( handle === '' ) return domain;
	return domain + '/' + handle;
}

function getQueryString(handle:string, domain:string) :string {
	let q = 'domain='+encodeURIComponent(domain);
	if( handle !== '' ) q += '&handle='+encodeURIComponent(handle)
	return q;
}