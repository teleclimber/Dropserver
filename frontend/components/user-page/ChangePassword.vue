<style scoped>
	.submit {
		display: flex;
		justify-content: space-between;
		margin-top:2em;
	}
	input[type="text"],
	input[type="password"] {
		height: 2rem;
		font-size: 1rem;
		padding: 0 0.2rem;
		margin: 0;
		box-sizing: border-box;
	}
	.input-grid {
		display: grid;
		grid-template-columns: 10rem 1fr 10rem;
		grid-column-gap: 0.5em;
		grid-row-gap: 0.5em;
	}
	.input-grid Label {
		grid-column: 1 / 2;
		justify-self: end;
		align-self: center;
	}
	.input-grid input {
		grid-column: 2/3;
	}
	.input-grid .error-msg {
		grid-column: 3 / 4;
		color: red;
		align-self: center;
	}
</style>

<template>
	<DsModal>
		<h2>Change Password</h2>
		<section class="input-grid">
			<label>Old password:</label>
			<input id="old_pw" type="password" v-model="change_pw_vm.old_pw"/>
			<span class="error-msg" v-if="change_pw_vm.validations.old_pw">{{change_pw_vm.validations.old_pw}}</span>

			<label>New password:</label>
			<input id="new_pw" type="password" v-model="change_pw_vm.new_pw" />
			<span class="error-msg" v-if="change_pw_vm.validations.new_pw">{{change_pw_vm.validations.new_pw}}</span>

			<label>New one again:</label>
			<input id="repeat_pw" type="password" v-model="change_pw_vm.repeat_pw" />
			<span class="error-msg" v-if="change_pw_vm.validations.repeat_pw">{{change_pw_vm.validations.repeat_pw}}</span>
		</section>

		<div class="submit">
			<DsButton @click="change_pw_vm.cancel()" type="cancel">Cancel</DsButton>
			<DsButton @click="change_pw_vm.doSave()" :disabled="!change_pw_vm.validations.valid">Save</DsButton>
		</div>
	</DsModal>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import ChangePwVM from '../../vms/user-page/change-pw-vm';
import { DataValidations } from '../../vms/user-page/change-pw-vm';

import DsModal from '../ui/DsModal.vue';
import DsButton from '../ui/DsButton.vue';

@Observer
@Component({
	components: {
		DsModal,
		DsButton
	}
})
export default class ChangePassword extends Vue {
	@Prop({required: true, type: ChangePwVM}) readonly change_pw_vm!: ChangePwVM;
}

</script>