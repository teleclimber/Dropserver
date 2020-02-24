<style scoped>
	section {
		_border: 2px solid grey;
		margin: 0 0 10px;
		padding: 10px;
		background-color: white;
	}
	.version {
		font-weight: normal;
		color: #888;
	}
	.paused {
		background-color: orange;
		color:white;
		font-weight: normal;
		font-size: 0.9rem;
		padding: 0.2rem 0.5rem;
		border-radius: 0.2rem;
	}
	.app-url {
		color: rgb(145, 145, 145);
		text-decoration: none;
		display: block;
		margin: 1em 0;
	}
	.app-url:hover {
		color: blue;
		text-decoration: underline;
	}
</style>

<template>
	<section>
		<h3>
			{{appspace_vm.application.app_name}}
			<span class="version">{{appspace_vm.appspace.app_version}}</span>
			<span class="paused" v-if="appspace_vm.appspace.paused">paused</span>
		</h3>
		<a :href="appspace_vm.open_url" class="app-url">
			{{appspace_vm.display_url}}
		</a>

		<span class="upgrade" v-if="appspace_vm.upgrade">
			Upgrade available: {{appspace_vm.upgrade}}
			<DsButton @click="user_page_vm.showUpgradeAppspace(appspace_vm.appspace.appspace_id)">upgrade</DsButton>
		</span>

		<DsButton @click="user_page_vm.showManageAppspace(appspace_vm.appspace.appspace_id)">manage</DsButton>
		
	</section>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import UserPageUI from "../../vms/user-page/user-page-ui";
import AppspaceVM from '../../vms/user-page/appspace-vm';

import DsButton from '../ui/DsButton.vue';

@Observer
@Component({
	components: {
		DsButton
	}
})
export default class AppspaceListItem extends Vue {
	@Inject(UserPageUI.injectKey) readonly user_page_vm!: UserPageUI;
	@Prop({required: true, type: AppspaceVM}) readonly appspace_vm!: AppspaceVM;
}
</script>
