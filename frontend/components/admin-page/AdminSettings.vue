<style scoped>
	.submit {
		display: flex;
		justify-content: space-between;
		margin-top:2em;
	}
</style>

<template>
	<div>
		<h2>Settings</h2>

		<div class="loading" v-if="!vm.orig_data">
			loading...
		</div>
		<template v-else>
			<p>
				Registration: 
				<select ref="registration_input" 
						:value="vm.orig_data.registration_open"
						@input="changed"
				>
					<option value="open">open to the public</option>
					<option value="closed">invitation only</option>
				</select>
			</p>

			<div class="submit">
				<DsButton @click="vm.close()" type="close">Close</DsButton>
				<DsButton @click="save" :disabled="vm.disable_save_btn">Save</DsButton>
			</div>
		</template>
	</div>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import DsButton from '../ui/DsButton.vue';

@Observer
@Component({
components: {
		DsButton
	}
})
export default class AdminSettings extends Vue {
	@Prop() readonly vm!: any;
	//@Inject('vm') readonly vm!: any;	// TODO

	@Ref('registration_input') readonly registration_input!: HTMLInputElement;
	
	collectData() {
		return {
			registration_open: this.registration_input.value
		}
	}
	changed() {
		this.vm.inputChanged( this.collectData() );
	}
	save() {
		this.vm.doSave( this.collectData() );
	}
};
</script>