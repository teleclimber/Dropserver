import axios from 'axios';
import {ax} from '../controllers/userapi';
import type {AxiosResponse, AxiosError} from 'axios';

// generic log data interface for app/apspace logs.

// type LogChunk struct {
// 	From    int64  `json:"from"`
// 	To      int64  `json:"to"`
// 	Content string `json:"content"`
// }

type LogChunk = {
	from: number,
	to: number,
	content: string
}

export class LiveLog {
	from = 0;
	to = 0;
	content = "";
	loaded = false;

	async initInProcessAppLog(appGetKey :string) {
		let resp :AxiosResponse|undefined;
		try {
			resp = await ax.get('/api/application/in-process/'+appGetKey+'/log');
		}
		catch(error: any | AxiosError) {
			throw error;
		}
		if( resp?.data === undefined ) return;

		const log_chunk = <LogChunk>resp.data;
		this.from = log_chunk.from;
		this.to = log_chunk.to;
		this.content = log_chunk.content;
		this.loaded = true;
	}

	async initAppspaceLog(appspaceID :number) {
		let resp :AxiosResponse|undefined;
		try {
			resp = await ax.get('/api/appspace/'+appspaceID+'/log');
		}
		catch(error: any | AxiosError) {
			throw error;
		}
		if( resp?.data === undefined ) return;

		const log_chunk = <LogChunk>resp.data;
		this.from = log_chunk.from;
		this.to = log_chunk.to;
		this.content = log_chunk.content;
		this.loaded = true;
	}
}
