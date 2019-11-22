module.exports = {
	preset: 'ts-jest',
	testEnvironment: 'jsdom',
	collectCoverage: true,
	collectCoverageFrom: [
		"**/*.{js,ts}",
		"!**/node_modules/**"
	],
	
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