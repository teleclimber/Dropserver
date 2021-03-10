module.exports = {
	preset: 'ts-jest',
	testEnvironment: 'jsdom',
	collectCoverage: false,
	collectCoverageFrom: [
		"**/*.{js,ts}",
		"!**/node_modules/**"
	],
};
