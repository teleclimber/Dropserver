import {reactive} from 'vue';
import {get} from '../controllers/userapi';

import {AppVersion} from './app_versions';

// apps, is basically a collection of app versions.

// From go:
// type App struct {
// 	ID       string    `json:"id" api:"apps"`
// 	Name     string    `json:"name" api:"attr"`
// 	Created  time.Time `json:"created_dt" api:"attr"`
// 	Versions []string  `json:"versions" api:"rel,app_versions,app"`
// 	//Owner    string    `json:"owner" api:"rel,users"`
// }

export class App {
	loaded = false;
	app_id = 0;
	name= '';
	created_dt = new Date;

	versions : AppVersion[] = [];	// wondering if thous should be some sort of collection, so that it can be sorted, filtered, etc...

	// fetch()...
	setFromRaw(raw :any) {
		this.app_id = Number(raw.app_id);
		this.name = raw.name + '';
		this.created_dt = new Date(raw.created_dt);

		if( Array.isArray(raw.versions) ) {
			this.versions = raw.versions.map((rawVer:any) => {
				const av = new AppVersion();
				av.setFromRaw(rawVer);
				return av;
			});
		}

		this.loaded = true;
	}
}

export class Apps {
	apps : Map<number,App> = new Map();

	async fetchForOwner() {
		const resp_data = await get('/application');
		resp_data.apps.forEach( (raw:any) => {
			const app = new App;
			app.setFromRaw(raw);
			this.apps.set(app.app_id, app);
		});
	}

	get asArray() : App[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		return Array.from(this.apps.values());
	}
}

export function ReactiveApps() {
	return reactive(new Apps);
}


// enum LoadStatus {
// 	Needed,
// 	Loading,
// 	Loaded
// }

// // not 100% sure we need a collector here?
// class ACollector {
// 	apps : Map<number,App> = new Map();
// 	status: Map<number, LoadStatus> = new Map();

// 	load_counter = 0;
// 	load_timeout = 0;

// 	get(app_id: number) :App {
// 		let app = this.apps.get(app_id);
// 		if( app !== undefined ) return app;
		
// 		app = reactive(new App);
// 		app.app_id = app_id;
		
// 		this.apps.set(app_id, app);
// 		this.status.set(app_id, LoadStatus.Needed);
		
// 		this.touch();

// 		return app;
// 	}
// 	touch() {
// 		window.clearTimeout(this.load_timeout);
// 		this.load_timeout = window.setTimeout(() => {
// 			this.loadNeeded();
// 		}, 200);
// 	}
// 	async loadNeeded() {
// 		console.log("loading app versions");
		
// 		const needed : string[] = [];
// 		this.status.forEach((status, id) => {
// 			if( status === LoadStatus.Needed ) {
// 				needed.push('id='+encodeURIComponent(id));
// 				this.status.set(id, LoadStatus.Loading);
// 			}
// 		});

// 		if( needed.length ) {
// 			const resp_data = await get('/application/?'+needed.join('&'));
// 			resp_data.app_versions.forEach((raw:any) =>{
// 				const id_string = idString(Number(raw.app_id), raw.version);
// 				const av = this.avs.get(id_string);
// 				if( av === undefined ) throw new Error("app version undefined after loading due to need.");
// 				av.setFromRaw(raw);
// 				this.status.set(id_string, LoadStatus.Loaded);
// 			});
// 		}
// 	}


// }

// export class Apps {

// 	apps : App[] = []

// 	async fetchForOwner() {
// 		const resp_data = await get('/apps?include=versions&filter=owner');
// 		const doc = new Document(resp_data);
// 		this.apps = doc.getCollection().map(res => {
// 			const app = new App;
// 			app.setFromResource(res);

// 			// const app_version_rels = res.relMany('versions');
// 			// app.versions = app_version_rels.map(res => {
// 			// 	const app_version = new AppVersion;
// 			// 	const inc_res = doc.getIncluded('app_versions', res.idString());
// 			// 	console.log("res / incres", res, inc_res);
// 			// 	app_version.setFromResource(inc_res);
// 			// 	return app_version;
// 			// });
			
			
// 			return app;
// 		});
// 	}
// }

// export function ReactiveApps() {
// 	return reactive(new Apps);
// }


// type AppVersion = {
// 	loading: true,	// could also do some sort of embedded type or whatever, if we want to expand on the concept.
// 	app_name: string,
// 	version: string,
// 	schema: number,
// 	created_dt: Date,
// }

// class AppVersions {
// 	apps_versions: Map<number,Map<string,AppVersion>> = reactive(new Map());

// 	getAppVersion(app_id:number, version:string) :AppVersion {
// 		const app_versions = this.apps_versions.get(app_id);
// 		if( app_versions !== undefined ) {
// 			const app_version = app_versions.get(version);
// 			if( app_version !== undefined ) return app_version;
// 		}

// 		return this.setAppVersion(app_id, version, {
// 			loading: true,
// 			app_name: '',
// 			version: 'x.x.x',
// 			created_dt: new Date(),
// 			schema: 0
// 		});
// 	}

// 	setAppVersion(app_id:number, version:string, av:AppVersion) :AppVersion {
// 		let app_versions = this.apps_versions.get(app_id);
// 		if( app_versions === undefined ) this.apps_versions.set(app_id, new Map);
// 		app_versions = <Map<string,AppVersion>>this.apps_versions.get(app_id);
// 		if( app_versions.has(version) ) {
// 			// merge new data in
// 			const app_version = app_versions.get(version);
// 			Object.assign(app_version, av);
// 		}
// 		else {
// 			app_versions.set(version, av);
// 		}
// 		return <AppVersion>app_versions.get(version);
// 	}
// }

// export const app_versions = new AppVersions();