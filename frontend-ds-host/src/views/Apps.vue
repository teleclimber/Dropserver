<template>
	<ViewWrap>
		<h1>Apps</h1>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

import ViewWrap from '../components/ViewWrap.vue';

export default defineComponent({
	name: 'Apps',
	components: {
		ViewWrap
	},
});
</script>