<style scoped>
	@layer base {
		@responsive {
			.ds-full-layout {
				display: grid;
				grid-template-columns: 16rem 1fr;
				grid-template-rows: 5.5rem 1fr;
				grid-template-areas:
					"nav header"
					"nav view";
			}
		}
	}
</style>

<template>
	<div class="md:ds-full-layout w-full md:h-screen">
		<NavMain></NavMain>
		<HeaderMain></HeaderMain>
		<div class="pt-16 md:pt-0 md:overflow-y-scroll">
			<router-view/>
		</div>
	</div>
	<UserDropDownMenu v-if="user_menu_open"></UserDropDownMenu>
</template>


<script lang="ts">
import { defineComponent } from "vue";

import NavMain from "./components/NavMain.vue";
import HeaderMain from "./components/HeaderMain.vue";
import UserDropDownMenu from './components/UserDropDownMenu.vue';

import { user_menu_open } from './controllers/nav';

export default defineComponent({
	name: "App",
	components: {
		NavMain,
		HeaderMain,
		UserDropDownMenu,
	},
	setup() {
		return {
			user_menu_open,
		}
	}
});
</script>

