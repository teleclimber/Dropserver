import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import CurrentUserDM from '../../dms/current-user-dm'
import ApplicationsUI from './applications-ui';

import ChangePwVM from './change-pw-vm';
import ApplicationsDM from '../../dms/applications-dm.js';
import AppspacesUI from './appspaces-ui.js';

type UserPageUIDeps = {
	current_user_dm: CurrentUserDM,
	applications_dm: ApplicationsDM,
	applications_ui: ApplicationsUI,
	appspaces_ui: AppspacesUI,
}

export default class UserPageUI {	// rename to UI
	static injectKey = Symbol();

	@observable change_pw_vm: ChangePwVM | undefined;

	constructor(private deps: UserPageUIDeps) {
		this.deps.applications_ui.parent = this;
	}

	@action
	closeAllModals() {
		// Shouldn't this all be internal to each vm?

		return true;
	}

	@action
	showCreateAppspace( app_id?: number, version?: string ) {
		this.deps.appspaces_ui.showCreate(app_id, version);
	}

	@action
	showManageAppspace( appspace_id: number ) {
		this.deps.appspaces_ui.showManage( appspace_id );
	}

	@action
	showUpgradeAppspace( appspace_id: number ) {
		this.deps.appspaces_ui.showUpgrade( appspace_id );
	}

	@action
	closeManageAppSpace() {
		//this.show_manage_appspace = false;
	}


	// manage applications:
	@action
	showApplicationsList() {
		this.deps.applications_ui.showList();	//make that a bit smarter please
	}

	// manage an application
	@action
	showManageApplication(app_id: number) {
		this.deps.applications_ui.showManageApplication(app_id);
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