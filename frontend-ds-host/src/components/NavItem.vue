<template>
	<li class="opacity-75 hover:opacity-100 hover:bg-blue-900">
		<router-link :to="to" class="block pl-4 py-6 text-white font-semibold" active-class="bg-blue-900"><slot></slot></router-link>
	</li>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'NavItem',
  components: {
    
  },
  props: ['to']
});
</script>