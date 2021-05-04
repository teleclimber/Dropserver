import {get, post, del} from '../controllers/userapi';

export type AppspaceBackup = {
	name: string,
	download_link: string,
}
export class AppspaceBackups {
	files :AppspaceBackup[] = [];

	loaded = false;

	constructor(private appspace_id:number) {}

	async fetchForAppspace() {
		const resp_data = await get('/appspace/'+this.appspace_id+'/export');
		resp_data.forEach( (raw:any) => {
			this.files.push(this.dataFromRaw(raw));
		});
		this.loaded = true;
	}
	dataFromRaw(raw:any) :AppspaceBackup {
		return {
			name: raw+'',
			download_link: '/api/appspace/'+this.appspace_id+'/export/'+raw
		};
	}
	async backupNow() {
		const resp_data = await post('/appspace/'+this.appspace_id+'/export/', {});
		this.files.unshift(this.dataFromRaw(resp_data.filename));
	}
	async delete(filename:string) {
		await del('/appspace/'+this.appspace_id+'/export/'+filename);
		const i = this.files.findIndex(f=> f.name === filename)
		if( i === -1 )return
		this.files.splice(i, 1);
	}
}
