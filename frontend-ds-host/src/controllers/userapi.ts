import {reactive } from 'vue';
import axios from 'axios';
import type {AxiosResponse} from 'axios';

type Err = {
	method: string,
	path: string,
	code:number,
	message: string
}
class ReqErrStack_ {
	errs: Err[] = [];

	push(err:Err) {
		this.errs.push(err);
	}
}
export const ReqErrStack = reactive(new ReqErrStack_);


import user from '../models/user';

const ax = axios.create();
ax.interceptors.response.use(function (response) {
		return response;
	}, function (error) {
		if( error.response && error.response.status >= 400 ) {
			const resp = error.response;
			if( resp.status == 401 ) user.setUnauthorized();
			else {
				ReqErrStack.push({method:resp.config.method, path:resp.config.url, code: resp.status, message: resp.data});
				console.log("pushed error to stack");
			}
		}

		return Promise.reject(error);
	});

const path_prefix = '/api';

const options = {
	headers: {'Content-Type': 'application/json'}
	// also add accept
};

// this handles sending requests to backend api and handles errors consistently.
// It's a controller because it interacts with parts of the page that warn user that they need to log in.

// it's possible this should actually be used by data models.

export async function get(path :string) :Promise<any> {	// string for now, we can get more fany with a getter object later.

	let resp:AxiosResponse;
	try {
		resp = await ax.get( path_prefix + path );
	}
	catch(e) {
		// handle error
		console.error('caught error in get request '+path, e);
		return;
	}

	return resp.data;
}

export async function patch(path:string, data:any) :Promise<any> {

	let resp:AxiosResponse;
	try {
		resp = await ax.patch( path_prefix + path, data, options );
	}
	catch(e) {
		// handle error
		console.error(e);
		return;
	}

	// if 401 then notify page user needs to log in.

	return resp.data;
}


export async function post(path:string, data:any) :Promise<any> {

	let resp:AxiosResponse;
	try {
		resp = await ax.post( path_prefix + path, data, options );
	}
	catch(e) {
		// handle error
		console.error(e);
		return;
	}

	// if 401 then notify page user needs to log in.

	return resp.data;
}