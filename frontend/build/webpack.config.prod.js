const webpack = require( 'webpack' );
const { VueLoaderPlugin } = require('vue-loader');
const HtmlWebpackPlugin = require( 'html-webpack-plugin' );

const path = require( 'path' );

module.exports = {
	mode: 'production',
	entry: { 
		admin: './views/admin/admin.js' ,
		user: './views/user/user.js'
	},
	output: {
		filename: '[name].js',
		path: path.join( process.cwd(), 'dist/static/' ),
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

		new VueLoaderPlugin(),

		// html:
		new HtmlWebpackPlugin({
			filename: '../resources/admin.html',
			template: 'views/admin/admin.html',
			chunks: ['admin'],
			inject: true
		}),
		new HtmlWebpackPlugin({
			filename: '../resources/user.html',
			template: 'views/user/user.html',
			chunks: ['user'],
			inject: true
		}),
	]
};