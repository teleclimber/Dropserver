module.exports = {
	preset: 'ts-jest',
	testEnvironment: 'jsdom',
	
	moduleFileExtensions: [
	  "js",
	  "ts",
	  "json",
	  "vue"
	],
	transform: {
	  ".*\\.(vue)$": "vue-jest"
	},
	globals: {
	  "vue-jest": {
		babelConfig: false
	  }
	}
  };