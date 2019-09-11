<style scoped>
	.header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}
	invitations-container {
		max-height: calc( 90vh - 6rem );
		overflow-y: scroll;
	}
	.invite {
		padding: 0.5rem;
		background-color: white;
		margin-bottom: 1px;
	}
	.invite:hover {
		background-color: #ffa;
		cursor: pointer;
	}
	.submit {
		display: flex;
		justify-content: space-between;
		margin-top:2em;
	}
</style>

<template>
	<div>
		<h2>User Invitations</h2>

		<div class="header">
			<p>{{vm.invitations.length}} invitations</p>
			<DsButton @click="vm.showPage('new')">New Invitation</DsButton>
		</div>

		<div class="invitations-container">
			<div class="invite"
				v-for="invitation in vm.invitations"
				:key="invitation.email"
				@click="vm.showPage('invitation', invitation.invitation)"
			>
				{{invitation.email}}
			</div>
		</div>

		<div class="submit">
			<DsButton @click="vm.close()" type="close">Close</DsButton>
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
export default class AdminInvitationsList extends Vue {
	@Prop() readonly vm!: any;	//todo: inject
};
</script>