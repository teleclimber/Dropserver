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
			<input id="old_pw" ref="old_pw" type="password" @input="inputChanged" />
			<span class="error-msg" v-if="change_pw_vm.validations.old_pw">{{change_pw_vm.validations.old_pw}}</span>

			<label>New password:</label>
			<input id="new_pw" ref="new_pw" type="password" @input="inputChanged" />
			<span class="error-msg" v-if="change_pw_vm.validations.new_pw">{{change_pw_vm.validations.new_pw}}</span>

			<label>New one again:</label>
			<input id="repeat_pw" ref="repeat_pw" type="password" @input="inputChanged" />
			<span class="error-msg" v-if="change_pw_vm.validations.repeat_pw">{{change_pw_vm.validations.repeat_pw}}</span>
		</section>

		<div class="submit">
			<DsButton @click="vm.closeChangePassword" type="cancel">Cancel</DsButton>
			<DsButton @click="doSave" :disabled="!change_pw_vm.validations.valid">Save</DsButton>
		</div>
	</DsModal>
</template>

<script>
import DsModal from '../../components/ds-modal.vue';
import DsButton from '../../components/ds-button.vue';

export default {
	name: 'ChangePassword',
	components: {
		DsModal,
		DsButton
	},
	computed: {
		vm: function() {
			return this.$root;
		},
		change_pw_vm: function() {
			return this.vm.change_pw_vm;
		}
	},
	methods: {
		collectData: function() {
			return {
				old_pw: this.$refs.old_pw.value,
				new_pw: this.$refs.new_pw.value,
				repeat_pw: this.$refs.repeat_pw.value
			};
		},
		inputChanged: function() {
			this.change_pw_vm.inputChanged( this.collectData() );
		},
		doSave: function() {
			this.change_pw_vm.doSave( this.collectData() );
		}
	}
}

</script>