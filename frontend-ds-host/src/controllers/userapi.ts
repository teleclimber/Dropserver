import axios from 'axios';
import type {AxiosResponse} from 'axios';

const ax = axios.create();
ax.interceptors.response.use(function (response) {
		return response;
	}, function (error) {
		if( error.response && error.response.status == 401 ) {
			alert('You are not logged in')
			window.location.href = '/login';
		}

		return Promise.reject(error);
	});

const path_prefix = '/api';

const options = {
	headers: {'Content-Type': 'application/vnd.api+json'}
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
		console.error(e);
		return;
	}

	// if 401 then notify page user needs to log in.

	return resp.data;

}

export async function patch(path:string, json:string) :Promise<any> {

	let resp:AxiosResponse;
	try {
		resp = await ax.patch( path_prefix + path, json, options );
	}
	catch(e) {
		// handle error
		console.error(e);
		return;
	}

	// if 401 then notify page user needs to log in.

	return resp.data;
}