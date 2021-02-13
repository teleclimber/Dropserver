import {get, patch} from '../controllers/userapi';
import { reactive } from 'vue';

// From go:
// type VersionMeta struct {
// 	AppID      domain.AppID      `json:"app_id"`
// 	AppName    string            `json:"app_name"`
// 	Version    domain.Version    `json:"version"`
// 	APIVersion domain.APIVersion `json:"api_version"`
// 	Schema     int               `json:"schema"`
// 	Created    time.Time         `json:"created_dt"`
// }

export class AppVersion {
	loaded = false;
	// need an "error" flag too.

	app_id = 0;
	version ='';
	app_name = '';
	api_version = 0;
	schema = 0;
	created_dt = new Date;


	async fetch(app_id: number, version: string) {
		const resp_data = await get('/application/'+app_id+'/version/'+version);
		this.setFromRaw(resp_data);
	}
	setFromRaw(raw :any) {
		this.app_id = Number(raw.app_id);
		this.version = raw.version+'';
		this.app_name = raw.app_name+'';
		this.api_version = Number(raw.api_version);
		this.schema = Number(raw.schema);
		this.created_dt = new Date(raw.created_dt);
		this.loaded = true;
	}
	
}

enum LoadStatus {
	Needed,
	Loading,
	Loaded
}

class AVCollector {
	avs : Map<string,AppVersion> = new Map();
	status: Map<string, LoadStatus> = new Map();

	load_counter = 0;
	load_timeout = 0;

	get(app_id: number, version: string) :AppVersion {
		const id_string = idString(app_id, version);
		let av = this.avs.get(id_string);
		if( av !== undefined ) return av;
		
		av = reactive(new AppVersion);	// is this necessary?
		av.app_id = app_id;
		av.version = version;
		
		this.avs.set(id_string, av);
		this.status.set(id_string, LoadStatus.Needed);
		
		this.touch();

		return av;
	}
	touch() {
		window.clearTimeout(this.load_timeout);
		this.load_timeout = window.setTimeout(() => {
			this.loadNeeded();
		}, 200);
	}
	async loadNeeded() {
		const needed : string[] = [];
		this.status.forEach((status, id) => {
			if( status === LoadStatus.Needed ) {
				needed.push('app-version='+encodeURIComponent(id));
				this.status.set(id, LoadStatus.Loading);
			}
		});

		if( needed.length ) {
			const resp_data = await get('/application/?'+needed.join('&'));
			resp_data.app_versions.forEach((raw:any) =>{
				const id_string = idString(Number(raw.app_id), raw.version);
				const av = this.avs.get(id_string);
				if( av === undefined ) throw new Error("app version undefined after loading due to need.");
				av.setFromRaw(raw);
				this.status.set(id_string, LoadStatus.Loaded);
			});
		}
	}


}

function idString(app_id:number, version:string) :string {
	return app_id+'-'+version;
}

export const AppVersionCollector = reactive(new AVCollector);
