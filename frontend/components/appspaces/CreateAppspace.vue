<style scoped>
	.action-pending {
		margin: 4em 0;
		text-align: center;
		color: #888;
		font-size: 1.2rem;
		font-style: italic;
	}
	.submit {
		display: flex;
		justify-content: space-between;
		margin-top:2em;
	}
</style>

<template>
	<DsModal>
		<h2>Create Appspace</h2>

		<div class="action-pending" v-if="create_vm.action_pending">
			{{create_vm.action_pending}} Make this dpendant on state and create multiple as needed.
		</div>
		<template v-else-if="create_vm.state === 'created'">
			<p>Created.</p>
			<p>
				<a :href="create_vm.created_appspace.open_url">
					{{ create_vm.created_appspace.display_url }}
				</a>
			</p>
			<div class="submit">
				<DsButton @click="create_vm.close()" type="close">Close</DsButton>
			</div>
		</template>
		<template v-else>
			Application: 
			<select v-model="create_vm.app_id">
				<option v-for="app in applications_dm.applications" :key="app.app_id" :value="app.app_id">{{app.app_name}}</option>
			</select>
			<select v-model="create_vm.version">
				<option v-for="v in create_vm.app_versions" :key="v.version" :value="v.version">{{v.version}}</option>
			</select>
			<p>{{create_vm.app_id}} {{create_vm.version}}</p>
			<!-- pick version OR specify auto-update/latest -->

			<div class="submit">
				<DsButton @click="create_vm.close()" type="cancel">cancel</DsButton>
				<DsButton @click="create_vm.create()" :disabled="!create_vm.inputs_valid">Create Appspace</DsButton>
			</div>
		</template>

		<!-- 
			pick app {optional bifurk to add application},
			..use latest version by default, but can select version in UI
			key/id selection / generation,
			[description] 
		
		 -->
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref, Watch } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import ApplicationsDM from '../../dms/applications-dm';

import { CreateAppspaceVM } from '../../vms/user-page/appspaces-vm';

import DsModal from '../ui/DsModal.vue';
import DsButton from '../ui/DsButton.vue';

@Observer
@Component({
	components: {
		DsModal,
		DsButton
	}
})
export default class CreateAppspace extends Vue {
	@Inject(ApplicationsDM.injectKey) readonly applications_dm!: ApplicationsDM;

	@Prop({required: true, type: CreateAppspaceVM}) readonly create_vm!: CreateAppspaceVM;
}
</script>