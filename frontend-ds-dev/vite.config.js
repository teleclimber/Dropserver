import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// https://vitejs.dev/config/
export default defineConfig({
	//root: '/dropserver-dev/',
	base: '/dropserver-dev/',
	server: {
		proxy: {
			'/dropserver-dev/avatar': {
				target: 'http://localhost:3003/',
				changeOrigin: true
			},
			'/dropserver-dev/base-data': {
				target: 'http://localhost:3003/',
				changeOrigin: true
			},
			'/dropserver-dev/livedata': {
				target: 'http://localhost:3003/',
				ws: true,
				changeOrigin: true,
				configure: (proxy, options) => {
					// proxy will be an instance of 'http-proxy'
					proxy.on('proxyReqWs', (proxyReq, req, res) => {
						console.log("proxy WS req for ws: changing origin con la mano");
						proxyReq.setHeader('Origin', "ws://localhost:3003");
						// Have to set Origin for websocket's check origin 
					})
				}
			},
		}
	},
	resolve: { alias: { '@': '/src' } },
	plugins: [vue()]
});
