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

<script>
import { observer } from "mobx-vue";

import DsButton from '../../../components/ds-button.vue';

export default observer({
	name: 'AdminInvitateNew',
	props: ['vm'],
	components: {
		DsButton
	},
	methods: {
		collectData: function() {
			return {
				email: this.$refs.email_input.value
			};
		},
		changed: function() {
			this.vm.inputChanged( this.collectData() );
		},
		save: function() {
			//collect data and send it over.
			this.vm.save( this.collectData() );
		}
	}
});
</script>