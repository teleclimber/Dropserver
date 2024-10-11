import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

const proxy_target = {
	protocol: 'http:',
    host: 'dropid.local.dropserver.org',
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
			"/events": prox
		}
	},
	resolve: { alias: { '@': '/src' } },
	plugins: [vue()]
});