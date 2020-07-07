import { action, computed, observable, decorate, configure, runInAction, intercept, observe } from "mobx";

import AppspacesDM from '../../dms/appspaces-dm';
import AppspaceDM from '../../dms/appspace-dm';
import ApplicationsDM from '../../dms/applications-dm';
import {VersionDM} from '../../dms/application-dm';
import LiveDataDM, {MigrationStatus, LiveMigrationJob} from '../../dms/live-data-dm';

import AppspaceVM from './appspace-vm';

type ManageAppspaceVMDeps = {
	appspaces_dm: AppspacesDM,
	applications_dm: ApplicationsDM,
	live_data_dm: LiveDataDM
}
type ManageAppspaceVMCbs = {
	closeClicked(): void
}

export enum ManageState { start, pick_version, show_upgrade };

export default class ManageAppspaceVM {
	@observable action_pending: string = '';// get rid of this
	@observable state: ManageState = ManageState.start;	//make that an EditState
	@observable appspace_vm: AppspaceVM;

	@observable upgrade_version: VersionDM | undefined;

	@observable delete_check: string = '';

	constructor(private cbs: ManageAppspaceVMCbs, private deps: ManageAppspaceVMDeps, private appspace_id: number) {
		const appspace = this.deps.appspaces_dm.getAppspace(appspace_id);
		this.appspace_vm = new AppspaceVM({
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
