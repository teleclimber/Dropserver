import {reactive } from 'vue';
import { useAuthUserStore } from '@/stores/auth_user';
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

export const ax = axios.create();
ax.interceptors.response.use(function (response) {
		return response;
	}, function (error) {
		if( error.response && error.response.status >= 400 ) {
			const resp = error.response;
			if( resp.status == 401 ) {
				const authUserStore = useAuthUserStore();
				authUserStore.setUnauthorized();
			}
			else if( resp.status != 404 ) {
				ReqErrStack.push({method:resp.config.method, path:resp.config.url, code: resp.status, message: resp.data});
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


// Maybe we don't need any of this, 
// Just export ax and have models consume directly (I think path prefix can be set in Axios)
export async function get(path :string) :Promise<any> {	// string for now, we can get more fany with a getter object later.
	const resp = await ax.get( path_prefix + path );
	return resp.data;
}

export async function patch(path:string, data:any) :Promise<any> {
	const resp = await ax.patch( path_prefix + path, data, options );
	return resp.data;
}
export async function patchForm(path:string, data:FormData) :Promise<any> {
	const resp = await ax.patch( path_prefix + path, data );
	return resp.data;
}

export async function post(path:string, data:any) :Promise<any> {
	let resp:AxiosResponse;
	try {
		resp = await ax.post( path_prefix + path, data, options );
	}
	catch(e) {
		console.error(e);
		return;
	}
	return resp.data;
}
export async function postForm(path:string, data:FormData) :Promise<any> {
	let resp:AxiosResponse;
	try {
		resp = await ax.post( path_prefix + path, data );
	}
	catch(e) {
		console.error(e);
		return;
	}
	return resp.data;
}

export async function del(path :string) :Promise<any> {	// string for now, we can get more fany with a getter object later.
	const resp = await ax.delete( path_prefix + path );
	return resp.data;
}