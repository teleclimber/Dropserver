<template>
	<div class="px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
		<dt class="font-medium text-gray-500 pt-2">
			{{field}}
		</dt>
		<dd class="mt-1 sm:mt-0 sm:col-span-2">
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