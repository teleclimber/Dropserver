import ds_axios from '../ds-axios-helper-ts';
import { AxiosResponse } from 'axios';

import { action, computed, observable, decorate, configure, runInAction, flow, observe } from "mobx";

import { compare as semverCompare, gt as semverGt, lt as semverLt } from 'semver';

type VersionData = {
	app_name: string,
	version: string,
	schema: number,
	created_dt: Date
}

type UploadVersionResp = {
	error: boolean,
	error_message?: string,
	version?: VersionDM
}

// PostVersionResp is
// type PostVersionResp struct {
// 	VersionMeta VersionMeta `json:"version_meta"`
// }
// type PostVersionResp = {
// 	version_meta: any
// }

export default class ApplicationDM {
	// AppID    int           `json:"app_id"`
	// AppName  string        `json:"app_name"`
	// Created  time.Time     `json:"created_dt"`
	// Versions []VersionMeta `json:"versions"`
	@observable app_id: number;
	@observable app_name: string;
	@observable created_dt: Date;
	@observable versions: VersionDM[];

	constructor(data:any) {
		this.app_id = Number(data.app_id);
		this.app_name = data.app_name+"";
		this.created_dt = new Date(data.created_dt);
		this.versions = [];
		if( Array.isArray(data.versions) ) {
			this.versions = data.versions.map( (v:any) => new VersionDM(v) );
		}
	}

	@computed get sorted_versions(): VersionDM[] {
		return this.versions.slice().sort( (a, b) => {
			return semverCompare(b.version, a.version);	// reverse order
		});
	}

	getVersion(version: string): VersionDM {
		const v = this.versions.find( (v:VersionDM) => v.version === version );
		if(!v) throw new Error('version not found');
		return v
	}
	versionExists(version: string): boolean {
		const v = this.versions.find( (v:VersionDM) => v.version === version );
		return !!v;
	}
	getPrevVersion(version:string): VersionDM | undefined {
		// it's the first one that is less than passed version
		const versions = this.sorted_versions;
		return versions.find( (v:VersionDM) => semverLt(v.version, version));
	}
	getNextVersion(version:string): VersionDM | undefined {
		const versions = this.sorted_versions;
		return versions.slice().reverse().find( (v:VersionDM) => semverGt(v.version, version));
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
			runInAction( () => {
				ret.version = new VersionDM(resp.data.version_meta);
				this.versions.push(ret.version);
			});	
		}

		return ret;
	}

}

// VersionMeta is for listing versions of application code
// type VersionMeta struct {
// 	AppName string         `json:"app_name"`
// 	Version domain.Version `json:"version"`
// 	Schema  int            `json:"schema"`
// 	Created time.Time      `json:"created_dt"`
// }
export class VersionDM {
	@observable app_name: string;
	@observable version: string;
	@observable schema: number;
	@observable created_dt: Date;

	constructor(data:any) {
		this.app_name = data.app_name+"";
		this.version = data.version+"";
		this.schema = Number(data.schema);
		this.created_dt = new Date(data.created_dt);
	}

}