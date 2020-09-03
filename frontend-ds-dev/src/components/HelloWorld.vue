<template>
	<div>
		<h1 class="text-purple-700">{{ msg }}</h1>
	</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'HelloWorld',
  props: {
	msg: String,
  },
});
</script>

