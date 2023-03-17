<template>
	<div class="px-4 sm:px-6 py-2 sm:grid sm:grid-cols-3 sm:gap-4 items-baseline">
		<dt class="font-medium text-gray-500 ">
			{{field}}
		</dt>
		<dd class="sm:col-span-2">
			<slot></slot>
		</dd>
	</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'DataDef',
  components: {
    
  },
  props: ['field']
});
</script>