import { action, computed, observable, decorate, configure, runInAction, intercept, observe } from "mobx";

import { ApplicationMeta, VersionMeta, MigrationStatusResp } from '../../generated-types/userroutes-classes';

import AppspacesDM from '../../dms/appspaces-dm';
import AppspaceDM from '../../dms/appspace-dm';
import ApplicationsDM from '../../dms/applications-dm';
import LiveDataDM, {MigrationStatus, LiveMigrationJob} from '../../dms/live-data-dm';

import AppspaceVM from './appspace-vm';

type AppspacesVMDeps = {
	appspaces_dm: AppspacesDM,
	applications_dm: ApplicationsDM,
	live_data_dm: LiveDataDM
}
// no parent on this one?
export default class AppspacesVM {
	static injectKey = Symbol();

	@observable create_appspace_vm: CreateAppspaceVM | undefined;
	@observable manage_appspace_vm: ManageAppspaceVM | undefined;

	constructor(private deps: AppspacesVMDeps){	}

	getAugmentedAppspace(appspace:AppspaceDM):AppspaceVM {
		return new AppspaceVM(this, {applications_dm: this.deps.applications_dm, live_data_dm: this.deps.live_data_dm}, appspace);
	}

	@action
	showCreate(app_id?: number, version?: string) {
		this.create_appspace_vm = new CreateAppspaceVM({
			getAugmentedAppspace: (appspace:AppspaceDM) => this.getAugmentedAppspace(appspace),
			closeClicked: () => this.create_appspace_vm = undefined
		}, {
			appspaces_dm: this.deps.appspaces_dm,
			applications_dm: this.deps.applications_dm,
			live_data_dm: this.deps.live_data_dm
		}, app_id, version);
	}
	@action
	closeCreateClicked() {
		this.create_appspace_vm = undefined;
	}

	@action
	showManage(appspace_id: number) {
		// create the vm with appspace data
		this.manage_appspace_vm = new ManageAppspaceVM({
			closeClicked: () => this.manage_appspace_vm = undefined
		}, {
			applications_dm: this.deps.applications_dm,
			appspaces_dm: this.deps.appspaces_dm,
			live_data_dm: this.deps.live_data_dm
		}, appspace_id );
	}
	@action
	showUpgrade(appspace_id: number) {
		this.showManage(appspace_id);

		const upgrade_ver = this.manage_appspace_vm?.appspace_vm.upgrade;
		if( upgrade_ver ) {
			this.manage_appspace_vm?.pickVersion(upgrade_ver);
		}
		else {
			this.manage_appspace_vm?.showPickVersion();
		}
	}
	@action
	closeManageClicked() {

	}

	@action
	pauseAppSpace( appspace_id:number, pause_on:boolean ) {
		// app_spaces_vm.action_pending = pause_on ? 'Pausing...' : 'Unpausing...';
		// ds_axios.patch( '/api/logged-in-user/appspaces/'+encodeURIComponent(app_space.id), {
		// 	pause: !!pause_on
		// } ).then( () => {
		// 	app_space.paused = !!pause_on;
		// 	app_spaces_vm.action_pending = null;
		// });
	}

}

// CreateAppspaceVM (dynamic)
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

export class CreateAppspaceVM {
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
			if( !change.newValue.find((v:VersionMeta) => v.version === this.version) ) {
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
	get app_versions(): VersionMeta[] {
		if( this.app_id == undefined ) return [];
		const a = this.deps.applications_dm.getApplication(this.app_id);
		return a.versions;
	}

	@computed
	get inputs_valid(): boolean {
		if( this.app_id == undefined ) return false;
		const app = this.deps.applications_dm.getApplication(this.app_id);
		
		if( !this.version ) return false;
		if( !app.versions.find((v:VersionMeta) => v.version === this.version) ) return false;
		
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

// break this out into its own file please.
// ManageAppspaceVM
type ManageAppspaceVMDeps = {
	appspaces_dm: AppspacesDM,
	applications_dm: ApplicationsDM,
	live_data_dm: LiveDataDM
}
type ManageAppspaceVMCbs = {
	closeClicked(): void
}

export enum ManageState { start, pick_version, show_upgrade };

export class ManageAppspaceVM {
	@observable action_pending: string = '';// get rid of this
	@observable state: ManageState = ManageState.start;	//make that an EditState
	@observable appspace_vm: AppspaceVM;

	@observable upgrade_version: VersionMeta | undefined;

	@observable delete_check: string = '';

	constructor(private cbs: ManageAppspaceVMCbs, private deps: ManageAppspaceVMDeps, private appspace_id: number) {
		const appspace = this.deps.appspaces_dm.getAppspace(appspace_id);
		this.appspace_vm = new AppspaceVM({
			showManage: function(){},	// TODO: why are these functions neutered? -> because appspace vm has them for an API when it really shouldn't.
			showUpgrade: function(){}
		}, {
			applications_dm: this.deps.applications_dm,
			live_data_dm: this.deps.live_data_dm
		}, appspace );
	}

	// version change:
	@action
	showPickVersion() {
		this.state = ManageState.pick_version;
	}

	@action
	pickVersion(version: string) {
		this.upgrade_version = this.appspace_vm.application.getVersion(version);
		this.state = ManageState.show_upgrade;
	}
	@computed get up_down(): string {
		if( this.upgrade_version === undefined ) return '...';
		const up_ver = this.upgrade_version.version;
		const cur_ver = this.appspace_vm.version.version;
		if( up_ver === cur_ver ) return '';

		const versions = this.appspace_vm.application.sorted_versions;
		const cur_i = versions.findIndex( v => v.version === cur_ver );
		const mig_i = versions.findIndex( v => v.version === up_ver );

		return cur_i > mig_i ? 'Upgrade' : 'Downgrade';
	}
	@computed get show_upgrade_btn() {
		return this.up_down !== '' && this.state === ManageState.show_upgrade;
	}

	doUpgrade() {
		// this goes to DM
		// This is what's next .... (I think backend is written now?) -> yes

		if( this.upgrade_version ) {
			// show a spinner or something while waiting?
			this.appspace_vm.appspace.changeVersion(this.upgrade_version);
		}
	}

	//pause 
	pause(pause:boolean) {
		// send that to dm
		this.appspace_vm.appspace.doPause(pause);
	}

	// delete
	@computed get allow_delete() {
		return this.delete_check === this.appspace_vm.subdomain;
	}
	doDelete() {
		//send to dm
	}

	//
	close() {
		this.cbs.closeClicked();
	}
}



