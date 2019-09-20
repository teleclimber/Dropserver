import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import CurrentUserDM from '../../dms/current-user-dm'
import ApplicationsVM from '../../vms/user-page/applications-vm';

import ChangePwVM from '../../vms/user-page/change-pw-vm';
import ApplicationsDM from '../../dms/applications-dm.js';
import AppspacesVM from './appspaces-vm.js';

type UserPageVMDeps = {
	current_user_dm: CurrentUserDM,
	applications_dm: ApplicationsDM,
	applications_vm: ApplicationsVM,
	appspaces_vm: AppspacesVM,
}

export default class UserPageVM {
	static injectKey = Symbol();

	@observable change_pw_vm: ChangePwVM | undefined;

	constructor(private deps: UserPageVMDeps) {
		this.deps.applications_vm.parent = this;
	}

	@action
	closeAllModals() {
		// Shouldn't this all be internal to each vm?

		return true;
	}

	@action
	showCreateAppspace( app_id?: number, version?: string ) {
		this.deps.appspaces_vm.showCreate(app_id, version);
	}

	@action
	showManageAppspace( app_space: any, shortcut: any ) {
		//this.appspaces_vm.manageAppSpace( app_space );
		if( shortcut ) {
			//if( shortcut.page === 'upgrade' ) this.appspaces_vm.showUpgradeVersion( shortcut.version );
		}
	}

	@action
	closeManageAppSpace() {
		//this.show_manage_appspace = false;
	}


	// manage applications:
	@action
	showApplicationsList() {
		this.deps.applications_vm.showList();	//make that a bit smarter please
	}

	// manage an application
	@action
	showManageApplication(app_id: number) {
		this.deps.applications_vm.showManageApplication(app_id);
	}
	@action
	cancelManageApplication() {
		// cancel at VM? Is this used?
	}

	//password
	@action
	showChangePassword() {
		this.change_pw_vm = new ChangePwVM(this, {current_user_dm: this.deps.current_user_dm});
	}
	@action
	closeChangePassword() {
		this.change_pw_vm = undefined;
	}


}