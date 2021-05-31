module.exports = {
	publicPath: '/dropserver-dev/',
	chainWebpack: (config) => {
		config
		.plugin('html')
		.tap((args) => {
			args[0].filename = 'index.html',
			args[0].template = 'index.html',
			//args[0].chunks: ['ds-dev'],
			args[0].inject = true
			//args[0].title = 'Custom Title';
			return args;
		});
	},
	devServer: {
		proxy: {
			'/dropserver-dev': {
				target: 'http://localhost:3003/',
				ws: true,
				changeOrigin: true,
				onProxyReqWs: (proxyReq, req, res) => {
					console.log("proxy WS req for ws: changing origin con la mano");
					proxyReq.setHeader('Origin', "ws://localhost:3003");
					// Have to set Origin for websocket's check origin 
				}
			}
		}
	}
}