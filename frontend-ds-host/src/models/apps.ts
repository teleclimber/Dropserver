import {reactive} from 'vue';

// lots here.
// Let's start by lazy loading app versions
// Let's make app-versions a separate listing from apps, yes?


type AppVersion = {
	loading: true,	// could also do some sort of embedded type or whatever, if we want to expand on the concept.
	app_name: string,
	version: string,
	schema: number,
	created_dt: Date,
}

class AppVersions {
	apps_versions: Map<number,Map<string,AppVersion>> = reactive(new Map());

	getAppVersion(app_id:number, version:string) :AppVersion {
		const app_versions = this.apps_versions.get(app_id);
		if( app_versions !== undefined ) {
			const app_version = app_versions.get(version);
			if( app_version !== undefined ) return app_version;
		}

		return this.setAppVersion(app_id, version, {
			loading: true,
			app_name: '',
			version: 'x.x.x',
			created_dt: new Date(),
			schema: 0
		});
	}

	setAppVersion(app_id:number, version:string, av:AppVersion) :AppVersion {
		let app_versions = this.apps_versions.get(app_id);
		if( app_versions === undefined ) this.apps_versions.set(app_id, new Map);
		app_versions = <Map<string,AppVersion>>this.apps_versions.get(app_id);
		if( app_versions.has(version) ) {
			// merge new data in
			const app_version = app_versions.get(version);
			Object.assign(app_version, av);
		}
		else {
			app_versions.set(version, av);
		}
		return <AppVersion>app_versions.get(version);
	}
}

export const app_versions = new AppVersions();