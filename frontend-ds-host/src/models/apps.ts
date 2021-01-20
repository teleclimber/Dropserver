import {reactive} from 'vue';
import {get} from '../controllers/userapi';
import {Document, Resource} from '../utils/jsonapi_utils';

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

	versions : AppVersion[] = [];

	setFromResource(r :Resource) {
		this.app_id = r.idNumber();
		this.name = r.attrString('name');
		this.created_dt = r.attrDate('created_dt');

		this.loaded = true;
	}
}

export class Apps {

	apps : App[] = []

	async fetchForOwner() {
		const resp_data = await get('/apps?include=versions&filter=owner');
		const doc = new Document(resp_data);
		this.apps = doc.getCollection().map(res => {
			const app = new App;
			app.setFromResource(res);

			const app_version_rels = res.relMany('versions');
			app.versions = app_version_rels.map(res => {
				const app_version = new AppVersion;
				const inc_res = doc.getIncluded('app_versions', res.idString());
				console.log("res / incres", res, inc_res);
				app_version.setFromResource(inc_res);
				return app_version;
			});
			
			
			return app;
		});
	}
}

export function ReactiveApps() {
	return reactive(new Apps);
}


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