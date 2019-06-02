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
						:value="vm.orig_data.registration"
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

<script>
import { observer } from "mobx-vue";

import DsButton from '../../../components/ds-button.vue';

export default observer({
	name: 'AdminSettings',
	props: ['vm'],
	components: {
		DsButton
	},
	methods: {
		collectData: function() {
			return {
				registration: this.$refs.registration_input.value
			}
		},
		changed: function() {
			this.vm.inputChanged( this.collectData() );
		},
		save: function() {
			this.vm.doSave( this.collectData() );
		}
	}
});
</script>