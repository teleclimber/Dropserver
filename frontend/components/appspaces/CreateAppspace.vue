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

		<div class="action-pending" v-if="app_spaces_vm.action_pending">
			{{app_spaces_vm.action_pending}}
		</div>
		<template v-else-if="app_spaces_vm.state === 'created'">
			<p>Created.</p>
			<p>
				<a :href="app_spaces_vm.getOpenUrl(app_spaces_vm.managed_app_space)">
					{{ app_spaces_vm.getDisplayUrl(app_spaces_vm.managed_app_space) }}
				</a>
			</p>
			<div class="submit">
				<DsButton @click="doClose" type="close">Close</DsButton>
			</div>
		</template>
		<template v-else>
			Application: 
			<select ref="app_select" v-model="app_spaces_vm.create_data.app_id">
				<option value=""> </option>
				<option v-for="app in applications" :key="app.app_id" :value="app.app_id">{{app.app_name}}</option>
			</select>
			<select ref="version_select" @input="versionChanged">
				<option v-for="version in app_versions" :key="version" :value="version">{{version}}</option>
			</select>
			<!-- pick version OR specify auto-update/latest -->

			<div class="submit">
				<DsButton @click="doClose" type="cancel">cancel</DsButton>
				<DsButton @click="createAppSpace" :disabled="!inputs_valid">Create App Space</DsButton>
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

import DsModal from '../ui/DsModal.vue';
import DsButton from '../ui/DsButton.vue';

@Component({
	components: {
		DsModal,
		DsButton
	}
})
export default class CreateAppSpace extends Vue {
	@Inject() readonly user_vm!: any;
	@Inject() readonly applications_vm!: any;
	@Inject() readonly app_spaces_vm!: any;

	inputs_valid: boolean = false;

	@Ref('version_select') readonly version_select!: HTMLInputElement;

	get applications() {
		return this.applications_vm.applications;
	}
	get app_versions() {
		if( !this.app_spaces_vm.create_data.app_id ) return [];
		else {
			const app = this.applications.find( (a: any) => a.app_id === this.app_spaces_vm.create_data.app_id );
			return app.versions.map( (v: any) => v.version );
		}
	}
	
	@Watch('app_spaces_vm.create_data.app_id')
	onAppIdChange() {
		this.$nextTick().then( this.inputsValid );
	}
	

	doClose() {
		this.user_vm.cancelCreateAppSpace();
	}

	versionChanged() {
		
	}
	inputsValid(): any {
		this.inputs_valid = false;
		console.log( 'checking inputs valid' );
		const app_id = Number(this.app_spaces_vm.create_data.app_id);
		const app = this.applications.find( (a: any) => a.app_id === app_id );	// string versus num?
		if( !app ) return false;
		const version = this.version_select.value;
		if( !version ) return false;
		if( !app.versions.find((v:any) => v.version === version) ) return false;
		
		console.log( 'inputs ARE valid' );
		this.inputs_valid = true;

		return {
			app_id,
			version
		};
	}
	createAppSpace() {
		const inputs = this.inputsValid();
		if( inputs ) this.app_spaces_vm.createAppSpace( inputs );
	}
}
</script>