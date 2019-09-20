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
		<h2>Manage Appspace</h2>
		<div class="action-pending" v-if="manage_vm.action_pending">
			{{manage_vm.action_pending}}
		</div>
		<template v-else>
			<template v-if="manage_vm.state === ManageState.pick_version">
				<p>{{manage_vm.appspace_vm.application.app_name}}, {{manage_vm.appspace_vm.subdomain}}.</p>
				<p>Pick version:</p>
				<div class="versions-container">
					<div 
							class="version"
							:class="{ current: version.version === manage_vm.appspace_vm.app_version }"
							v-for="(version,i) in manage_vm.appspace_vm.application.versions"
							:key="version.version"
							@click="manage_vm.pickVersion(version.version)">
						<span class="ver-name">{{version.version}}</span>
						<span class="latest" v-if="i===0">latest</span>
						<span class="current" v-if="version.version === manage_vm.appspace_vm.app_version">current</span>
						<!-- could show latest version(?), number of app-spaces -->
					</div>
				</div>
			</template>
			<template v-else-if="manage_vm.state === ManageState.show_upgrade">
				<p>{{manage_vm.appspace_vm.application.app_name}}, {{manage_vm.appspace_vm.subdomain}}</p>
				<p>{{manage_vm.up_down}} from {{manage_vm.appspace_vm.version.version}} to {{manage_vm.upgrade_version.version}}</p>
				<p v-if="manage_vm.appspace_vm.version.schema !== manage_vm.upgrade_version.schema">
					Data migration necessary:
					from {{manage_vm.appspace_vm.version.schema}} to
					{{manage_vm.upgrade_version ? manage_vm.upgrade_version.schema : '...' }}
				</p>
				<p v-else>
					No Data migration necessary.
				</p>
			</template>
			<template v-else>
				<p v-if="manage_vm.appspace_vm.paused">
					Appspace is paused
					<DsButton @click="manage_vm.pause(false)">Unpause</DsButton>
				</p>
				<p v-else>
					Pause Appspace
					<DsButton @click="manage_vm.pause(true)">pause</DsButton>
				</p>
				<p>Subdomain: {{manage_vm.appspace_vm.subdomain}} [change?]</p>
				<p>Application: {{manage_vm.appspace_vm.application.app_name}}
					{{manage_vm.appspace_vm.version.version}}
					(data schema {{manage_vm.appspace_vm.version.schema}})
					<DsButton @click="manage_vm.showPickVersion()">Change version</DsButton>
				</p>
				
				<div class="delete">
					<p>Enter Address to delete:
					<input type="text" ref="del_check" class="del-check" v-model="manage_vm.delete_check">
					<DsButton @click="manage_vm.doDelete()" :disabled="!manage_vm.allow_delete">Delete</DsButton></p>
				</div>
			</template>

			<!-- upgrade app version, export data, delete appspace, archive, pause, clone, ... -->


			<div class="submit">
				<DsButton @click="manage_vm.close()" type="close">Close</DsButton>
				<DsButton @click="manage_vm.doUpgrade()" v-if="manage_vm.state === ManageState.show_upgrade">{{manage_vm.up_down}}</DsButton>
			</div>
		</template>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref, Watch } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import { ManageAppspaceVM, ManageState } from '../../vms/user-page/appspaces-vm';

import DsButton from '../ui/DsButton.vue';
import DsModal from '../ui/DsModal.vue';

@Observer
@Component({
	components: {
		DsModal,
		DsButton
	}
})
export default class ManageAppspace extends Vue {
	@Prop({required: true, type: ManageAppspaceVM}) readonly manage_vm!: ManageAppspaceVM;
	ManageState = ManageState;
}
</script>