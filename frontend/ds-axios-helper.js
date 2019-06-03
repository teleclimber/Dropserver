import axios from 'axios';

const ds_axios = axios.create({
	baseURL: window.ds_user_routes_base_url
});

export default ds_axios