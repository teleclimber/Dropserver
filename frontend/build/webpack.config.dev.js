const webpack = require( 'webpack' );
const { VueLoaderPlugin } = require('vue-loader');
const HtmlWebpackPlugin = require( 'html-webpack-plugin' );

const path = require( 'path' );

module.exports = {
	mode: 'development',
	entry: { 
		user:	['./views/user/user.ts', 'webpack-hot-middleware/client'],
		admin:	['./views/admin/admin.ts', 'webpack-hot-middleware/client'],
	},
	devtool: 'inline-source-map',
	output: {
		filename: '[name].js',
		path: path.join( process.cwd(), 'dist/views' ),
		publicPath: '/'
	},
	module: {
		rules: [{
			test: /\.vue$/,
			use: 'vue-loader'
		},{
			test: /\.tsx?$/,
			loader: 'ts-loader',
			exclude: /node_modules/,
			options: {
				appendTsSuffixTo: [/\.vue$/],
			}
		},{
			test: /\.css$/,
			use: [ 'vue-style-loader', 'css-loader' ]
		}]
	},
	resolve: {
		extensions: [ '.tsx', '.ts', '.js', '.vue'],
		alias: {
			'vue$': 'vue/dist/vue.esm.js'
		}
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
		hot: true,
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
			"/signup": {
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