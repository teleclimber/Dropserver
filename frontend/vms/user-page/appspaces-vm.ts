import { action, computed, observable, decorate, configure, runInAction, intercept, observe } from "mobx";

import { ApplicationMeta, VersionMeta } from '../../generated-types/userroutes-classes';

import AppspacesDM from '../../dms/appspaces-dm';
import AppspaceDM from '../../dms/appspace-dm';
import ApplicationsDM from '../../dms/applications-dm';

import AppspaceVM from './appspace-vm';

type AppspacesVMDeps = {
	appspaces_dm: AppspacesDM,
	applications_dm: ApplicationsDM,
}
// no parent on this one?
export default class AppspacesVM {
	static injectKey = Symbol();

	@observable create_appspace_vm: CreateAppspaceVM | undefined;
	@observable manage_appspace_vm: ManageAppspaceVM | undefined;

	constructor(private deps: AppspacesVMDeps){	}

	getAugmentedAppspace(appspace:AppspaceDM):AppspaceVM {
		return new AppspaceVM(this, {applications_dm: this.deps.applications_dm}, appspace);
	}

	@action
	showCreate(app_id?: number, version?: string) {
		this.create_appspace_vm = new CreateAppspaceVM({
			closeClicked: () => this.create_appspace_vm = undefined
		}, {
			appspaces_dm: this.deps.appspaces_dm,
			applications_dm: this.deps.applications_dm
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
			appspaces_dm: this.deps.appspaces_dm
		}, appspace_id );
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
}
type CreateAppspaceVMCbs = {
	closeClicked(): void
}
export class CreateAppspaceVM {
	@observable state: string = "";	// TODO: make this like EditState

	@observable action_pending: string | undefined;

	@observable app_id: number | undefined;
	@observable version: string | undefined;

	@observable created_appspace: AppspaceVM | undefined;

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
		
		this.action_pending = 'Creating...';	// make that na edit state.

		await this.deps.appspaces_dm.create(this.app_id, this.version);

		this.action_pending = undefined;
		//app_spaces_vm.state = 'created';
		// not sure . Maybe show a readout of metadata and a button to open the appspace?
		
	}

	close() {
		this.cbs.closeClicked();
	}
}

// ManageAppspaceVM
type ManageAppspaceVMDeps = {
	appspaces_dm: AppspacesDM,
	applications_dm: ApplicationsDM,

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
		this.appspace_vm = new AppspaceVM({showManage: function(){}}, {
			applications_dm: this.deps.applications_dm
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
		if( !this.upgrade_version ) return '...';
		const cur_version = this.appspace_vm.version;
		const versions = this.appspace_vm.application.versions;
		const cur_i = versions.findIndex( v => v === cur_version );
		const mig_i = versions.findIndex( v => v === this.upgrade_version );
		return cur_i > mig_i ? 'Upgrade' : 'Downgrade';	//version array is sorted backwards
	}

	doUpgrade() {
		// this goes to DM
		// backend isn't written for that yet so let's not worry about it for now.

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



