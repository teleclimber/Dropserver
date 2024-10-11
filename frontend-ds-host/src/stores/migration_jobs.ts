import { reactive, ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import { on } from '../sse';
import { LoadState, AppspaceMigrationJob } from './types';

function migrationJobFromRaw(raw :any) :AppspaceMigrationJob {
	return {
		job_id: Number(raw.job_id),
		owner_id: Number(raw.owner_id),
		appspace_id: Number(raw.appsapce_id),
		to_version: raw.to_version + "",
		created: new Date(raw.created),
		started: raw.started ? new Date(raw.started) : null,
		finished: raw.finished ? new Date(raw.finished) : null,
		priority: !!raw.priority,
		error: raw.error ? raw.error+"" : null,
	};
}

export const useAppspaceMigrationJobsStore = defineStore('appspace-migration-jobs', () => {
	const load_state :Map<number,LoadState> = reactive(new Map);

	const jobs :ShallowRef<Map<number,ShallowRef<AppspaceMigrationJob>>> = shallowRef(new Map());

	function isLoaded(appspace_id: number) {
		const l = load_state.get(appspace_id);
		return l === undefined ? false : l === LoadState.Loaded;
	}

	async function loadData(appspace_id: number) {
		const l = load_state.get(appspace_id);
		if( !l ) {
			load_state.set(appspace_id, LoadState.Loading);
			const resp = await ax.get('/api/migration-job?appspace_id='+appspace_id);
			if( !Array.isArray(resp.data) ) throw new Error("expected array for migration jobs");
			resp.data.forEach( (raw:any) => {
				setReplaceJob(migrationJobFromRaw(raw));
			});
			load_state.set(appspace_id, LoadState.Loaded);
		}
	}
	function setReplaceJob(job:AppspaceMigrationJob) {
		const job_id = job.job_id;
		const ex_job = jobs.value.get(job_id);
		if( ex_job === undefined ) {
			jobs.value.set(job_id, shallowRef(job));
			jobs.value = new Map(jobs.value);
		}
		else {
			ex_job.value = job;
		}
		return mustGetJob(job_id);
	}
	async function reloadData(appspace_id: number) {	// TODO this should become unnecessary with proper events from the data model
		const l = load_state.get(appspace_id);
		if( l === LoadState.Loading ) return;	// its' already loading so don't reload
		load_state.delete(appspace_id);
		await loadData(appspace_id);
	}

	on('MigrationJob', (raw) => {
		setReplaceJob(migrationJobFromRaw(raw));
	});

	function getJob(job_id: number) {
		return jobs.value.get(job_id);
	}
	function mustGetJob(job_id:number) {
		const j = getJob(job_id);
		if( j === undefined ) throw new Error("expected job id to exist");
		return j;
	}

	function getRunningAppspaceJobs(appspace_id: number) {
		return computed( () => {
			const ongoing = Array.from(jobs.value.values()).filter( j => 
				j.value.appspace_id === appspace_id 
				&& j.value.started
				&& !j.value.finished);
			ongoing.sort( (a,b) => {
				if( !a.value.started && !b.value.started ) return 0;
				if( !a.value.started ) return -1;
				if( !b.value.started ) return 1;
				return a.value.started.getTime() - b.value.started.getTime();
			});
			return ongoing;
		});
	}

	async function createMigrationJob(appspace_id:number, to_version:string) :Promise<ShallowRef<AppspaceMigrationJob>> {
		const resp = await ax.post('/api/migration-job', {appspace_id, to_version});
		return setReplaceJob(migrationJobFromRaw(resp.data));
	}

	return { isLoaded, loadData, reloadData, createMigrationJob, getJob, mustGetJob, getRunningAppspaceJobs };
});