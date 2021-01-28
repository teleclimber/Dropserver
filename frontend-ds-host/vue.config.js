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
				target: "https://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/signup": {
				target: "https://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/api": {
				target: "https://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/twine": {
				target: "https://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost",
				ws: true,
				onProxyReqWs: (proxyReq, req, res) => {
					console.log("proxy WS req for ws: changing origin con la mano");
					proxyReq.setHeader('Origin', "ws://user.dropserver.develop:3000");
					// Have to set Origin for websocket's check origin 
					// Note that "changeOrigin" acutally sets the Host field, not the "Origin".
				}
			}
		}
	}
}