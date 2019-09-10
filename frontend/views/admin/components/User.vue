<style scoped>
.user-container {
	background-color: white;
	padding: 0.5rem;
	border-bottom: 1px solid #eee;
	margin-bottom: 0.2rem;
}
</style>

<template>
	<div class="user-container">
		{{user.email}}
		<!-- button @click="user.dingItUp()">Ding it up</button -->
		<!-- button @click="user.del">Delete</button -->
	</div>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject } from "vue-property-decorator";
import { Observer } from "mobx-vue";

@Observer
@Component
export default class User extends Vue {
	@Prop() user!: any;	// TODO: turn that into proper User type
};
</script>