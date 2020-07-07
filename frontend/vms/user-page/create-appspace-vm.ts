import { action, computed, observable, decorate, configure, runInAction, intercept, observe } from "mobx";

import AppspacesDM from '../../dms/appspaces-dm';
import AppspaceDM from '../../dms/appspace-dm';
import ApplicationsDM from '../../dms/applications-dm';
import {VersionDM} from '../../dms/application-dm';
import LiveDataDM, {MigrationStatus, LiveMigrationJob} from '../../dms/live-data-dm';

import AppspaceVM from './appspace-vm';

type CreateAppspaceVMDeps = {
	appspaces_dm: AppspacesDM,
	applications_dm: ApplicationsDM,
	live_data_dm: LiveDataDM
}
type CreateAppspaceVMCbs = {
	getAugmentedAppspace(appspace:AppspaceDM): AppspaceVM
	closeClicked(): void
}

export enum CreateState { start, creating, migrating, done };

export default class CreateAppspaceVM {
	@observable state: CreateState = CreateState.start;

	@observable app_id: number | undefined;
	@observable version: string | undefined;

	@observable created_appspace: AppspaceVM | undefined;
	@observable _job: LiveMigrationJob | undefined;

	constructor(private cbs: CreateAppspaceVMCbs, private deps: CreateAppspaceVMDeps, app_id?: number, version?: string) {
		// ensure app_id is either a number or undefined.
		intercept(this, 'app_id', change => {
			if( !change.newValue && change.newValue !== 0 ) change.newValue = undefined;
			else change.newValue = Number(change.newValue);
			return change;
		});

		// set version to undefined if not found in app_versions
		observe(this, 'app_versions', change => {
			if( !change.newValue.find((v:VersionDM) => v.version === this.version) ) {
				runInAction( () => this.version = undefined );
			}
		});

		if( app_id != undefined ) {
			this.app_id = app_id;
		}
		if( version != undefined ) {
			this.version = version;
		}
	}

	@computed
	get app_versions(): VersionDM[] {
		if( this.app_id == undefined ) return [];
		const a = this.deps.applications_dm.getApplication(this.app_id);
		return a.versions;
	}

	@computed
	get inputs_valid(): boolean {
		if( this.app_id == undefined ) return false;
		const app = this.deps.applications_dm.getApplication(this.app_id);
		
		if( !this.version ) return false;
		if( !app.versions.find((v:VersionDM) => v.version === this.version) ) return false;
		
		return true;
	}

	@action
	async create() {
		if( !this.inputs_valid ) return;
		if( !this.app_id || !this.version ) return;	// appease the TS gods.
		
		this.state = CreateState.creating

		const created_resp = await this.deps.appspaces_dm.create(this.app_id, this.version);

		const job_id = created_resp.job_id;
		console.log("created appspace, job id", job_id);

		runInAction( () => {
			this.created_appspace = this.cbs.getAugmentedAppspace(created_resp.appspace);

			this.state = CreateState.migrating;
			this._job = this.deps.live_data_dm.getJob(job_id+'');
			if( this._job.status !== MigrationStatus.finished ) {
				this.state = CreateState.done;
			}
			else {
				const disposer = observe(this._job, "status", change => {
					console.log("status changed", change.newValue);
					if( change.newValue === MigrationStatus.finished ) {
						this.state = CreateState.done;
						disposer();
					}
				});
			}
		});

		

		

		// I want to straight up get the job from live data, even if it hasn't started yet
		//this._job = this.deps.live_data_dm.getJob(job_id+'');

		// const as_id = created_resp.appspace.appspace_id;
		// if( this.deps.live_data_dm.running[as_id] ) {
		// 	const job = this.deps.live_data_dm.running[as_id];
		// 	if( job.status != MigrationStatus.finished ) {
		// 		runInAction( () => this.state = CreateState.migrating );
		// 	}
		// }

		// // maybe an autorun or watch thing?
		// const disposer = observe(this.deps.live_data_dm, 'running', change => {
		// 	console.log("observing appspace live data", change.newValue);
		// 	if( change.newValue && change.newValue[as_id] && this.state === CreateState.creating ) {
		// 		console.log('state to migrating');
		// 		runInAction( () => this.state = CreateState.migrating );
		// 	} else if( this.state === CreateState.migrating ) {
		// 		// was migrating, no longer, so done
		// 		console.log('state to done');
		// 		runInAction( () => this.state = CreateState.done );
		// 		disposer();
		// 	}
		// });
	}

	close() {
		this.cbs.closeClicked();
	}

	@computed get migration_job() :LiveMigrationJob|undefined {
		if( this._job && !this._job._is_dummy ) return this._job;
		return undefined;
	}
}
