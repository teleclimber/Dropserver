module.exports = {
  publicPath: '/dropserver-dev/',
  chainWebpack: (config) => {
    config
      .plugin('html')
      .tap((args) => {
        args[0].filename = 'ds-dev.html',
			  args[0].template = 'ds-dev.html',
			  //args[0].chunks: ['ds-dev'],
			  args[0].inject = true
        //args[0].title = 'Custom Title';
        return args;
      });
  },
  devServer: {
    proxy: {
      '/dropserver-dev': {
        target: 'http://127.0.0.1:3003/'
      }
    }
  }
}