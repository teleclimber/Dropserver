<style scoped>
	.header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}
	.submit {
		display: flex;
		justify-content: space-between;
		margin-top:2em;
	}
</style>

<template>
	<div>
		<h2>New Invitation</h2>

		<p>
			Email: 
			<input type="text" ref="email_input" @input="changed" />
			<span class="email-exists" v-if="vm.exists">Already invited!</span>
		</p>

		<div class="submit">
			<DsButton @click="vm.close()" type="cancel">Cancel</DsButton>
			<DsButton @click="save" :disabled="vm.disable_save_btn">Save</DsButton>
		</div>

	</div>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import DsButton from '../../../components/ds-button.vue';

@Observer
@Component({
	components: {
		DsButton
	}
})
export default class AdminInvitateNew extends Vue {
	@Prop() readonly vm!: any;	//todo: inject

	@Ref('email_input') readonly email_input!: HTMLInputElement;

	collectData() {
		return {
			email: this.email_input.value
		};
	}
	changed() {
		this.vm.inputChanged( this.collectData() );
	}
	save() {
		//collect data and send it over.
		this.vm.save( this.collectData() );
	}
};
</script>