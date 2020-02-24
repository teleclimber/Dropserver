import ds_axios from '../ds-axios-helper-ts';
import { AxiosResponse } from 'axios';

import { action, computed, observable, decorate, configure, runInAction, flow, observe } from "mobx";
import { ApplicationMeta, VersionMeta, PostVersionResp } from '../generated-types/userroutes-classes';

import autoDecorate from '../utils/mobx-auto-decorate';
autoDecorate(ApplicationMeta);

import { compare as semverCompare, gt as semverGt, lt as semverLt } from 'semver';

type UploadVersionResp = {
	error: boolean,
	error_message?: string,
	version_meta?: VersionMeta
}

export default class ApplicationDM extends ApplicationMeta {

	constructor(data:ApplicationMeta) {
		super(data);
	}

	@computed get sorted_versions(): VersionMeta[] {
		return this.versions.slice().sort( (a, b) => {
			return semverCompare(b.version, a.version);	// reverse order
		});
	}

	getVersion(version: string): VersionMeta {
		const v = this.versions.find( (v:VersionMeta) => v.version === version );
		if(!v) throw new Error('version not found');
		return v
	}
	versionExists(version: string): boolean {
		const v = this.versions.find( (v:VersionMeta) => v.version === version );
		return !!v;
	}
	getPrevVersion(version:string): VersionMeta | undefined {
		// it's the first one that is less than passed version
		const versions = this.sorted_versions;
		return versions.find( (v:VersionMeta) => semverLt(v.version, version));
	}
	getNextVersion(version:string): VersionMeta | undefined {
		const versions = this.sorted_versions;
		return versions.slice().reverse().find( (v:VersionMeta) => semverGt(v.version, version));
	}

	async uploadNewVersion(selected_files: SelectedFile[]): Promise<UploadVersionResp> {
		let ret: UploadVersionResp= {error: false};
		let resp: AxiosResponse<any>;

		const form_data = new FormData();
		selected_files.forEach((sf)=> {
			form_data.append( 'app_dir', sf.file, sf.rel_path );
		});

		try {
			resp = await ds_axios.post( '/api/application/'+encodeURIComponent(this.app_id)+'/version/', form_data, {	
				headers: {
					'Content-Type': 'multipart/form-data'
				},
				validateStatus: status => status == 200 || status == 422
			});
		}
		catch(e) {
			ret.error = true;
			return ret;
		}

		if( resp.status == 422 ) {
			ret.error = true;
			ret.error_message = resp.data.error;
		}
		else {
			ret.error = false;

			const resp_inst = new PostVersionResp(resp.data);

			runInAction( () => {
				this.versions.push(resp_inst.version_meta);
			});
			
			ret.version_meta = resp_inst.version_meta;
		}

		return ret;
	}

}