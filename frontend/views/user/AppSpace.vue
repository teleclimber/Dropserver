<style scoped>
	section {
		_border: 2px solid grey;
		margin: 0 0 10px;
		padding: 10px;
		background-color: white;
	}
	.version {
		font-weight: normal;
		color: #888;
	}
	.paused {
		background-color: orange;
		color:white;
		font-weight: normal;
		font-size: 0.9rem;
		padding: 0.2rem 0.5rem;
		border-radius: 0.2rem;
	}
	.app-url {
		color: rgb(145, 145, 145);
		text-decoration: none;
		display: block;
		margin: 1em 0;
	}
	.app-url .app-id {
		__color: black;
	}
	.app-url:hover {
		color: blue;
		text-decoration: underline;
	}
</style>

<template>
	<section>
		<h3>
			{{application.app_name}}
			<span class="version">{{app_space.app_version}}</span>
			<span class="paused" v-if="app_space.paused">paused</span>
		</h3>
		<a :href="open_url" class="app-url">
			{{display_url}}
		</a>

		<span class="upgrade" v-if="upgrade">
			Upgrade available: {{upgrade}}
			<DsButton @click="doUpgrade">upgrade</DsButton>
		</span>

		<DsButton @click="manage">manage</DsButton>
		
	</section>
</template>

<script>

import DsButton from '../../components/ds-button.vue';

export default {
	name: 'AppSpace',
	props: ['app_space'],
	computed: {
		open_url: function() {
			return this.$root.app_spaces_vm.getOpenUrl( this.app_space );
		},
		display_url: function() {
			return this.$root.app_spaces_vm.getDisplayUrl( this.app_space );
		},
		application: function() {
			let a = this.$root.applications_vm.applications.find( a => a.app_id === this.app_space.app_id );
			if( a ) {
				return a
			}
			return {
				versions:[]
			}
		},
		upgrade: function() {
			if( this.application && this.application.versions.length != 0 ) {
				const latest_version = this.application.versions[0];
				if( latest_version.name !== this.app_space.app_version ) {
					return latest_version.name;
				}
			}
		}
	},
	components: {
		DsButton
	},
	methods: {
		manage: function() {
			this.$root.showManageAppSpace( this.app_space );
		},
		doUpgrade: function() {
			this.$root.showManageAppSpace( this.app_space, { page:'upgrade', version: this.upgrade } );
			// probably some sub-part of manage app space?
		}
	}
}
</script>
