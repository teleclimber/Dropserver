import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import ApplicationsDM from '../../dms/applications-dm';
import ApplicationDM, {VersionDM} from '../../dms/application-dm';// is that really a DM?
import AppspacesDM from "../../dms/appspaces-dm";
import {VersionComparison} from './app-uses-vm';

import SelectFilesVM from '../ui/select-app-files-vm';
import AppUsesVM from "./app-uses-vm";


export enum EditState { start, upload, uploading, processing, error, enter_meta, finishing, finished };

type ManageApplicationVMDeps = {
	applications_dm: ApplicationsDM,
	appspaces_dm: AppspacesDM
}
type ManageApplicationVMCbs = {
	close(): void,
}
export default class ManageApplicationVM {
	app_id: number;
	application: ApplicationDM;
	app_uses: AppUsesVM;
	@observable show_version: VersionDM | undefined;
	@observable state: EditState = EditState.start;

	@observable delete_check: string = '';

	@observable select_files_vm: SelectFilesVM | undefined;

	constructor(private cbs: ManageApplicationVMCbs, private deps: ManageApplicationVMDeps, app_id: number) {
		this.app_id = app_id;	// why do we also need app_id on this?
		this.application = this.deps.applications_dm.getApplication(app_id);
		this.app_uses = new AppUsesVM(this.application, {appspaces_dm: this.deps.appspaces_dm} );
	}

	@action
	showVersion(version: VersionDM) {
		this.show_version = version;
	}

	@action 
	closeClicked() {
		if( this.show_version ) this.show_version = undefined;
		else this.cbs.close();
	}

	@action
	showVersionUpload() {
		this.select_files_vm = new SelectFilesVM;
		this.state = EditState.upload;
	}

	@computed get app_files_error(): string {
		if( !this.select_files_vm ) {
			return '';
		}
		if( this.select_files_vm.error ) {
			return this.select_files_vm.error;
		}

		if( !this.select_files_vm.app_files || !this.select_files_vm.metadata ) {
			return '';
		}

		const upload_version = this.select_files_vm.metadata.version;
		if( this.application.versionExists(upload_version) ) {
			return 'Version '+upload_version+' already exists';
		}

		return '';
	}

	@computed get version_comparison(): VersionComparison | undefined {
		if( !this.select_files_vm || !this.select_files_vm.app_files || !this.select_files_vm.metadata || this.app_files_error ) {
			return undefined;
		}

		const upload_version = this.select_files_vm.metadata.version;

		const ret:VersionComparison = {
			upload: this.select_files_vm.metadata,
			previous: this.application.getPrevVersion(upload_version),
			next: this.application.getNextVersion(upload_version),
			fatal: false,
			errors: {
				schema: '',
				version: ''
			}
		};

		if( (ret.previous && ret.previous.schema > ret.upload.schema)
			|| (ret.next && ret.next.schema < ret.upload.schema) ) {
			ret.fatal = true;
			ret.errors.schema = 'Schema is out of whack';
			return ret;
		}

		return ret;
	}

	@computed get enable_upload() {
		return this.select_files_vm 
		&& this.select_files_vm.app_files
		&& this.select_files_vm.metadata
		&& !this.select_files_vm.error
		&& this.version_comparison
		&& !this.version_comparison.fatal;
	}

	async uploadNewVersion() {
		if( !this.enable_upload ) return;
		if( !this.select_files_vm || !this.select_files_vm.app_files ) {	// ugh typescript
			return;
		}

		this.state = EditState.uploading;

		const upRet = await this.application.uploadNewVersion(this.select_files_vm.app_files);

		if( upRet.error || upRet.version == undefined ) {
			// I don't know what to do exactly.
			this.state = EditState.error;
		}
		else {
			// check upRet structure
			this.state = EditState.finished;

			// TODO: still somewhat unresolved what happens here
			//this.manage_status.app_meta = upRet.app_meta;
			//this.manage_status.version_meta = upRet.app_meta.versions[0];

			// I think what we have to do here is in flux, 
			// ..based on our incomplete implementation and even incomplete design of this aspect of DS

			// we could go back to manage app, which should show new version in listing?
			// ..or we need to show appsapces that could use an upgrade?
		}

	}

	async deleteVersion(version: string) {
		await this.deps.applications_dm.deleteVersion( this.application.app_id, version );
		if( this.show_version && this.show_version.version === version ) {
			runInAction( () => this.show_version = undefined );
		}
	}

	// delete 
	@computed get allow_delete() {
		return this.delete_check === this.application.app_name;
	}
	async deleteApplication() {
		await this.deps.applications_dm.deleteApplication( this.app_id );
		this.cbs.close();
	}

}