import { reactive, shallowRef, ShallowRef, triggerRef } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import { LoadState, UserIDProxyIDConflicts } from './types';
import { on } from '../sse';
import { userIDProxyIDConflictsFromRaw } from './appspaces';

function mapFromRaw(raw:any) {
	const c_map :Map<string, UserIDProxyIDConflicts> = new Map;
	for( let p in raw ) {
		const c = userIDProxyIDConflictsFromRaw(raw[p]);
		if( c ) c_map.set(p, c);
	}
	return c_map;
}

export const useAppspaceUserConflictsStore = defineStore('appspace-user-conflicts', () => {
	const appspace_user_conflicts : Map<number,ShallowRef<Map<string, UserIDProxyIDConflicts>>> = new Map();

	on('AppspaceUsers', (raw) => {
		const as_id = Number(raw);
		if( appspace_user_conflicts.has(as_id) ) loadData(as_id);
	});

	async function loadData(appspace_id: number) {
		const resp = await ax.get('/api/appspace/'+appspace_id+'/user-conflicts');
		const sr = getCreateAppspace(appspace_id);
		sr.value = mapFromRaw(resp.data);
		triggerRef(sr);
	}
	function getCreateAppspace(appspace_id :number) {
		const a = appspace_user_conflicts.get(appspace_id);
		if( a === undefined ) appspace_user_conflicts.set(appspace_id, shallowRef(new Map));
		return appspace_user_conflicts.get(appspace_id)!;
	}
	function getForAppspace(appspace_id: number) {
		if( !appspace_user_conflicts.has(appspace_id) ) loadData(appspace_id);
		return getCreateAppspace(appspace_id);
	}

	return {
		getForAppspace
	}
});