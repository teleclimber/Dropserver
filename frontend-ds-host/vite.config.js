import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

const proxy_target = {
	protocol: 'https:',
    host: 'dropid.dropserver.develop',
    port: 5050,
}

const prox = {
	target: proxy_target,
	changeOrigin: true,
	cookieDomainRewrite: ".localhost",
	secure: false,
}

// https://vitejs.dev/config/
export default defineConfig({
	clearScreen: false,
	build: {
		assetsDir: 'frontend-assets/',
	},
	server: {
		proxy: {
			"/login": prox,
			"/logout": prox,
			"/signup": prox,
			"/static": prox,
			"/appspacelogin": prox,
			"/api": prox,
			"/twine/": {
				target: "wss://dropid.dropserver.develop:5050",
				ws: true,
				changeOrigin: true,
				cookieDomainRewrite: ".localhost",
				secure: false,
				configure: (proxy, options) => {
					proxy.on("proxyReqWs", (proxyReq, req, res) => {
						console.log("proxy WS req for ws: changing origin con la mano");
						proxyReq.setHeader('Origin', "ws://dropid.dropserver.develop:5050");
					});
				}
			},
		}
	},
	resolve: { alias: { '@': '/src' } },
	plugins: [vue()]
});