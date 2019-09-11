const webpack = require( 'webpack' );
const { VueLoaderPlugin } = require('vue-loader');
const HtmlWebpackPlugin = require( 'html-webpack-plugin' );

const path = require( 'path' );

module.exports = {
	mode: 'production',
	entry: { 
		admin: './pages/admin/admin.ts' ,
		user: './pages/user/user.ts'
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
		extensions: ['.ts', '.js', '.vue', '.json'],
		alias: {
			'vue$': 'vue/dist/vue.esm.js'
		}
	},
	plugins:[

		new VueLoaderPlugin(),

		// html:
		new HtmlWebpackPlugin({
			filename: '../resources/admin.html',
			template: 'pages/admin/admin.html',
			chunks: ['admin'],
			inject: true
		}),
		new HtmlWebpackPlugin({
			filename: '../resources/user.html',
			template: 'pages/user/user.html',
			chunks: ['user'],
			inject: true
		}),
	]
};