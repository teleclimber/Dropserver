import { reactive, ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import twineClient from '../twine-services/twine_client';
import {SentMessageI} from 'twine-web';
import { LoadState, AppspaceMigrationJob } from './types';
import { useAppspacesStore } from './appspaces';

const remoteService = 12;

const remoteSubscribe = 11;
const remoteAppspaceSubscribe = 12;
const remoteUnsubscribe = 13;

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
	const connections :Map<number, SentMessageI|undefined> = reactive(new Map);

	const migration_jobs : ShallowRef<Map<number,ShallowRef<Map<number,ShallowRef<AppspaceMigrationJob>>>>> = shallowRef(new Map());

	function isLoaded(appspace_id: number) {
		const l = load_state.get(appspace_id);
		return l === undefined ? false : l === LoadState.Loaded;
	}

	async function loadData(appspace_id: number) {
		const l = load_state.get(appspace_id);
		if( !l ) {	// || l === LoadState.NotLoaded ) {
			load_state.set(appspace_id, LoadState.Loading);
			const resp = await ax.get('/api/migration-job?appspace_id='+appspace_id);
			if( !Array.isArray(resp.data) ) throw new Error("expected array for migration jobs");
			const jobs :Map<number,ShallowRef<AppspaceMigrationJob>> = new Map;
			resp.data.forEach( (raw:any) => {
				const j = migrationJobFromRaw(raw);
				jobs.set(j.job_id, shallowRef(j));
			});
			migration_jobs.value.set(appspace_id, shallowRef(jobs));
			migration_jobs.value = new Map(migration_jobs.value);
			load_state.set(appspace_id, LoadState.Loaded);
		}
	}

	function connected(appspace_id:number) :boolean {
		return connections.has(appspace_id);
	}
	async function connect(appspace_id:number) {
		if( !isLoaded(appspace_id) )throw new Error("wait until jobs are loaded to connect");
		if( connected(appspace_id) ) return;

		connections.set(appspace_id, undefined);

		const payload = new TextEncoder().encode(JSON.stringify({appspace_id}));

		await twineClient.ready();
		const subMessage = await twineClient.twine.send(remoteService, remoteAppspaceSubscribe, payload);
		connections.set(appspace_id, subMessage);

		const jobs = mustGetJobs(appspace_id);

		for await (const m of subMessage.incomingMessages()) {
			switch (m.command) {
				case 11:
					const raw :any = JSON.parse(new TextDecoder('utf-8').decode(m.payload));
					const in_job = migrationJobFromRaw(raw);
					const job_id = in_job.job_id;
					const ex_job = jobs.value.get(job_id);
					if( ex_job === undefined ) {
						jobs.value.set(job_id, shallowRef(in_job));
						jobs.value = new Map(jobs.value);
					}
					else {
						ex_job.value = in_job;
					}
					m.sendOK();
					break;
			
				default:
					m.sendError("What is this command?");
					throw new Error("what is this command? "+m.command);
			}
		}
	}
	async function disconnect(appspace_id:number) {
		const subMessage = connections.get(appspace_id);
		if( subMessage === undefined ) return;
		connections.set(appspace_id, undefined);
		await subMessage.refSendBlock(remoteUnsubscribe, undefined);
		connections.delete(appspace_id);
	}

	function getJobs(appspace_id:number) {
		if( isLoaded(appspace_id) ) return migration_jobs.value.get(appspace_id);
	}
	function mustGetJobs(appspace_id:number) {
		const jobs = getJobs(appspace_id);
		if( jobs === undefined ) throw new Error("expected appspace jobs to exist");
		return jobs;
	}

	async function createMigrationJob(appspace_id:number, to_version:string) :Promise<ShallowRef<AppspaceMigrationJob>> {
		const jobs = mustGetJobs(appspace_id);
		const resp = await ax.post('/api/migration-job', {appspace_id, to_version});
		const job = shallowRef(migrationJobFromRaw(resp.data));
		jobs.value.set(job.value.job_id, job);
		jobs.value = new Map(jobs.value);
		
		return job;
	}

	return { isLoaded, loadData, createMigrationJob, getJobs, mustGetJobs, connect, disconnect };
});