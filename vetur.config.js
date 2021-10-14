// vetur.config.js
// see https://vuejs.github.io/vetur/guide/setup.html#advanced

/** @type {import('vls').VeturConfig} */
module.exports = {
	settings: {
		"vetur.useWorkspaceDependencies": false,
		"vetur.experimental.templateInterpolationService": true,
		"vetur.validation.template": true,
		"vetur.validation.interpolation": true
	},
	projects: [
		{
			root: './frontend-ds-dev',
		},{
			root: './frontend-ds-host'
		}
	]
}
  