import { action, computed, observable, decorate, configure, runInAction, intercept, observe } from "mobx";

import AppspacesDM from '../../dms/appspaces-dm';
import AppspaceDM from '../../dms/appspace-dm';
import ApplicationsDM from '../../dms/applications-dm';
import LiveDataDM, {MigrationStatus, LiveMigrationJob} from '../../dms/live-data-dm';

import AppspaceVM from './appspace-vm';
import CreateAppspaceVM from './create-appspace-vm';
import ManageAppspaceVM from './manage-appspace-vm';

type AppspacesUIDeps = {
	appspaces_dm: AppspacesDM,
	applications_dm: ApplicationsDM,
	live_data_dm: LiveDataDM
}
// no parent on this one?
// So is this derived data or ui management?
// Judging by the methods, it's UI.
// I think I separated it out of user page because maybe the admin or other pages may need these functions?
export default class AppspacesUI {
	static injectKey = Symbol();

	@observable create_appspace_vm: CreateAppspaceVM | undefined;
	@observable manage_appspace_vm: ManageAppspaceVM | undefined;

	constructor(private deps: AppspacesUIDeps){	}

	getAugmentedAppspace(appspace:AppspaceDM):AppspaceVM {
		return new AppspaceVM( {applications_dm: this.deps.applications_dm, live_data_dm: this.deps.live_data_dm}, appspace);
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



