import axios from 'axios';

declare global {
	interface Window {
		ds_user_routes_base_url: string
	}
}

export const url = (window as Window).ds_user_routes_base_url

const ds_axios = axios.create({
	baseURL: url
});

export default ds_axios;

//export url;