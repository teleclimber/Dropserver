<style scoped>

</style>

<template>
	<component 
		:is="vm.page.is"
		:vm="vm.page"
	></component>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import AdminInvitationsList from './AdminInvitationsList.vue';
import AdminInviteNew from './AdminInviteNew.vue';
import AdminUpdateInvitation from './AdminUpdateInvitation.vue';

@Observer
@Component({
	components:{
		AdminInvitationsList,
		AdminInviteNew,
		AdminUpdateInvitation
	}
})
export default class AdminInvitations extends Vue {
	@Prop() readonly vm!: any;
	// todo: make that injected
};
</script>