// these are owner's appspaces, not remotes.
import {ref, reactive} from 'vue';
import axios from 'axios';

// hierarchical data:
// - appspaces (listing)
// - appspace (node of above list)
// - appspace status (derived live data on readiness of appspace)
// - appspace upgrade available 

// relations:
// - appspace -> appversion
// - appspace =>* contacts

// For relations (contacts here), we need:
// - related contact ids
// - data for these ids.
//   ..here should be a preview

type Appspace = {
	appspace_id: number,
	app_id: number,
	app_version: string,
	subdomain: string,
	created_dt: Date,
	paused: boolean,
}


class Appspaces {
	appspaces : Map<number,Appspace> = reactive(new Map());
	all_loaded = ref(false);

	async fetchAll() {
		if( this.all_loaded.value ) return;

		let resp:any;
		try {
			resp = await axios.get( '/api/appspace' );
		}
		catch(e) {
			// handle error
			console.error(e, resp);
		}

		if( !resp || !resp.data || !resp.data.appspaces ) return;

		// Since this is fetch all, reset the map completely.
		this.appspaces.forEach((_, id)=> this.appspaces.delete(id));

		const appspaces_resp = <Appspace[]>resp.data.appspaces ;
		appspaces_resp.forEach(as => {
			this.appspaces.set(Number(as.appspace_id), appspaceResp(as))
		});

		this.all_loaded.value = true;
	}

	get asArray() : Appspace[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		if( this.all_loaded.value ) return Array.from(this.appspaces.values());
		return [];
	}
}

function appspaceResp(data:any) :Appspace {
	return {
		appspace_id: Number(data.appspace_id),
		app_id: Number(data.app_id),
		app_version: data.app_version+'',
		subdomain: data.subdomain+'',
		paused: !!data.paused,
		created_dt: new Date(data.created_dt)
	}
}

export default new Appspaces;