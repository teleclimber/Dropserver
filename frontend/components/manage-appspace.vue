<style scoped>
	.submit {
		display: flex;
		justify-content: space-between;
		margin-top:2em;
	}
	.action-pending {
		margin: 4em 0;
		text-align: center;
		color: #888;
		font-size: 1.2rem;
		font-style: italic;
	}
	.version {
		padding: 0.8rem 10px;
		border-bottom: 1px solid #ddd;
		background-color: white;
		cursor: pointer;
	}
	.version:hover {
		background-color: #ffa;
	}
	.version.current {
		background-color: #ddd;
		color: #888;
		cursor: default;
	}
	.version .ver-name {
		font-weight: bold;
	}
	input.del-check {
		height: 2rem;
		padding: 0 0.2rem;
		margin: 0;
		box-sizing:border-box;
	}
</style>

<template>
	<DsModal>
		<h2>Manage App Space</h2>
		<div class="action-pending" v-if="app_spaces_vm.action_pending">
			{{app_spaces_vm.action_pending}}
		</div>
		<template v-else>
			<template v-if="app_spaces_vm.state === 'pick-version'">
				<p>{{application.app_name}}, {{app_space.subdomain}}.</p>
				<p>Pick version:</p>
				<div class="versions-container">
					<div 
							class="version"
							:class="{ current: version.version === app_space.app_version }"
							v-for="(version,i) in app_versions"
							:key="version.version"
							@click="pickVersion(version.version)">
						<span class="ver-name">{{version.version}}</span>
						<span class="latest" v-if="i===0">latest</span>
						<span class="current" v-if="version.version === app_space.app_version">current</span>
						<!-- could show latest version(?), number of app-spaces -->
					</div>
				</div>
			</template>
			<template v-else-if="app_spaces_vm.state === 'show-upgrade'">
				<p>{{application.app_name}}, {{app_space.subdomain}}</p>
				<p>{{up_down}} from {{cur_app_version.version}} to {{app_spaces_vm.upgrade_version}}</p>
				<p v-if="cur_app_version.schema !== migrate_ver_data.schema">
					Data migration necessary:
					from {{cur_app_version ? cur_app_version.schema : '...' }} to
					{{migrate_ver_data ? migrate_ver_data.schema : '...' }}
				</p>
				<p v-else>
					No Data migration necessary.
				</p>
			</template>
			<template v-else>
				<p v-if="app_space.paused">
					App space is paused
					<DsButton @click="pause(false)">Unpause</DsButton>
				</p>
				<p v-else>
					Pause App Space
					<DsButton @click="pause(true)">pause</DsButton>
				</p>
				<p>Address: {{app_space.id}} [change?]</p>
				<p>Application: {{application.app_name}}, v{{cur_app_version.version}}, data schema {{cur_app_version.schema}}
					<DsButton @click="$root.app_spaces_vm.showPickVersion">Change version</DsButton>
				</p>
				
				<div class="delete">
					<p>Enter Address to delete:
					<input type="text" ref="del_check" class="del-check" @input="delCheckInput">
					<DsButton @click="doDelete" :disabled="!allow_delete">Delete</DsButton></p>
				</div>
			</template>

			<!-- upgrade app version, export data, delete appspace, archive, pause, clone, ... -->


			<div class="submit">
				<DsButton @click="doClose" type="close">Close</DsButton>
				<DsButton @click="doUpgrade" v-if="app_spaces_vm.state === 'show-upgrade'">{{up_down}}</DsButton>
			</div>
		</template>
	</DsModal>
</template>

<script>
import DsButton from './ds-button.vue';
import DsModal from './ds-modal.vue';

export default {
	name: 'ManageAppSpace',
	data: function() {
		return {
			allow_delete: false
		};
	},
	components: {
		DsModal,
		DsButton
	},
	computed: {
		app_space: function() {
			return this.$root.app_spaces_vm.managed_app_space;
		},
		app_spaces_vm: function() {
			return this.$root.app_spaces_vm;
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
		app_versions: function() {
			if( this.application ) {
				return this.application.versions;
			}
			else return [];
		},
		cur_app_version: function() {
			return this.app_versions.find( v => v.version === this.app_space.app_version )
		},
		migrate_ver_data: function() {
			return this.app_versions.find( v => v.version === this.app_spaces_vm.upgrade_version );
		},
		up_down: function() {
			if( !this.app_spaces_vm.upgrade_version ) return;
			const cur_i = this.app_versions.findIndex( v => v.version === this.app_space.app_version );
			const mig_i = this.app_versions.findIndex( v => v.version === this.app_spaces_vm.upgrade_version );
			return cur_i > mig_i ? 'Upgrade' : 'Downgrade';	//version array is sorted backwards
		}
	},
	methods: {
		delCheckInput: function() {
			this.allow_delete = this.$refs.del_check.value.toLowerCase() === this.app_space.id.toLowerCase();
			return this.allow_delete;
		},
		doClose: function() {
			// close if that's allowable.
			if( this.app_spaces_vm.state === 'show-upgrade' ) this.app_spaces_vm.closeUpgradeVersion();
			else if( this.app_spaces_vm.state === 'pick-version' ) this.app_spaces_vm.closePickVersion();
			else this.$root.closeManageAppSpace();
		},
		doDelete: function() {
			if( this.delCheckInput() ) this.$root.app_spaces_vm.deleteAppSpace( this.app_space );
		},
		pause: function( pause_on ) {
			this.$root.app_spaces_vm.pauseAppSpace( this.app_space, pause_on );
		},
		pickVersion: function( version ) {
			if( version !== this.app_space.app_version ) this.$root.app_spaces_vm.showUpgradeVersion( version );
		},
		doUpgrade: function() {
			this.$root.app_spaces_vm.doUpgradeVersion();
		}
	}
}
</script>