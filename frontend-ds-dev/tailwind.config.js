module.exports = {
	mode: "jit",
	purge: [
		'./src/**/*.vue',
		'*.html'
	],
	future: {
		removeDeprecatedGapUtilities: true,
		purgeLayersByDefault: true,
	},
	theme: {
	  extend: {},
	},
	variants: {},
	plugins: [],
}