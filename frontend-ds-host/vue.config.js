module.exports = {
  publicPath: '/',
  devServer: {
		hot: true,
		proxy: {
			"/login": {
				target: "https://dropid.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/logout": {
				target: "https://dropid.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/signup": {
				target: "https://dropid.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/api": {
				target: "https://dropid.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/twine": {
				target: "https://dropid.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost",
				ws: true,
				onProxyReqWs: (proxyReq, req, res) => {
					console.log("proxy WS req for ws: changing origin con la mano");
					proxyReq.setHeader('Origin', "ws://dropid.dropserver.develop:3000");
					// Have to set Origin for websocket's check origin 
					// Note that "changeOrigin" acutally sets the Host field, not the "Origin".
				}
			}
		}
	}
}