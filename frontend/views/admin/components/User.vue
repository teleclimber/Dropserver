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

<script>
import { observer } from "mobx-vue";

export default observer({
	name: 'User',
	props: ['user']
});
</script>