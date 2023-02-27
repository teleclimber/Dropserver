<template>
	<div class="md:ds-full-layout w-full md:h-screen">
		<NavMain></NavMain>
		<HeaderMain></HeaderMain>
		<div class="pt-16 md:pt-0 bg-gray-100 md:overflow-y-scroll">
			<router-view/>
		</div>
	</div>
	<UserDropDownMenu v-if="user_menu_open"></UserDropDownMenu>
	<UserLoggedOutOverlay v-if="!authUserStore.logged_in"></UserLoggedOutOverlay>
	<ReqErrorOverlay></ReqErrorOverlay>
</template>


<script lang="ts">
import { defineComponent } from "vue";

import NavMain from "./components/NavMain.vue";
import HeaderMain from "./components/HeaderMain.vue";
import UserDropDownMenu from './components/UserDropDownMenu.vue';
import UserLoggedOutOverlay from './components/UserLoggedOutOverlay.vue';
import ReqErrorOverlay from './components/RequestErrorOverlay.vue';

import { user_menu_open } from './controllers/nav';
import { useAuthUserStore } from "./stores/auth_user";

export default defineComponent({
	name: "App",
	components: {
		NavMain,
		HeaderMain,
		UserDropDownMenu,
		UserLoggedOutOverlay,
		ReqErrorOverlay
	},
	setup() {
		const authUserStore = useAuthUserStore();
		return {
			user_menu_open,
			authUserStore
		}
	}
});
</script>

