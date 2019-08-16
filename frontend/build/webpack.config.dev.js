const webpack = require( 'webpack' );
const { VueLoaderPlugin } = require('vue-loader');
const HtmlWebpackPlugin = require( 'html-webpack-plugin' );

const path = require( 'path' );

module.exports = {
	mode: 'development',
	entry: { 
		user:	['./views/user/user.js', 'webpack-hot-middleware/client'],
		admin:	['./views/admin/admin.js', 'webpack-hot-middleware/client'],
	},
	output: {
		filename: '[name].js',
		path: path.join( process.cwd(), 'dist/views' ),
		publicPath: '/'
	},
	module: {
		rules: [{
			test: /\.vue$/,
			use: 'vue-loader'
		}, {
			test: /\.css$/,
			use: [ 'vue-style-loader', 'css-loader' ]
		}]
	},
	plugins:[

		new webpack.DefinePlugin({
			'window.ds_user_routes_base_url': JSON.stringify('//localhost:8080')	// clarify that this is the same as webpack devserver
		}),
		
		new VueLoaderPlugin(),

		// html:
		
		new HtmlWebpackPlugin({
			filename: 'admin.html',
			template: 'views/admin/admin.html',
			chunks: ['admin'],
			inject: true
		}),
		new HtmlWebpackPlugin({
			filename: 'user.html',
			template: 'views/user/user.html',
			chunks: ['user'],
			inject: true
		}),

		new webpack.HotModuleReplacementPlugin()
	],
	devServer: {
		proxy: {
			"/login": {
				target: "http://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/logout": {
				target: "http://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			},
			"/api": {
				target: "http://user.dropserver.develop:3000",
				changeOrigin: true,
				cookieDomainRewrite: ".localhost"
			}
		}
	}
};