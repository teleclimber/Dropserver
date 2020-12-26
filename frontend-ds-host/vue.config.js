module.exports = {
  publicPath: '/',
  // chainWebpack: (config) => {
  //   config
  //     .plugin('html')
  //     .tap((args) => {
  //       args[0].filename = 'index.html',
	// 		  args[0].template = 'index.html',
	// 		  //args[0].chunks: ['ds-dev'],
	// 		  args[0].inject = true
  //       //args[0].title = 'Custom Title';
  //       return args;
  //     });
  // },
  devServer: {
		hot: true,
		proxy: {
			"/login": {
				target: "https://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/logout": {
				target: "http://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/signup": {
				target: "http://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/api": {
				target: "http://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/live/**": {
				target: "ws://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost",
				ws: true,
				onProxyReqWs: (proxyReq, req, res) => {
					console.log("proxy WS req for ws: changing origin con la mano");
					proxyReq.setHeader('Origin', "ws://user.dropserver.develop:3000");
					// Have to set Origin for websocket's check origin 
					// Note that "changeOrigin" acutally sets the Host field, not the "Origin".
				}
			},
			"/live": {
				target: "http://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost",
			}
		}
	}
}