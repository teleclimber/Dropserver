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
		<h2>Update Invitation</h2>

		<p>Email: {{vm.invitation.email}}</p>

		<p>
			Delete: 
			<DsButton @click="vm.del()">Delete</DsButton>
		</p>

		<div class="submit">
			<DsButton @click="vm.close()" type="cancel">Cancel</DsButton>
		</div>

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
export default class AdminUpdateInvitation extends Vue {
	@Prop() readonly vm!: any;	//todo: inject
}
</script>