<template>
	<ViewWrap>
		<p>admin...</p>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

import ViewWrap from '../../components/ViewWrap.vue';

export default defineComponent({
  name: 'AdminHome',
  components: {
	  ViewWrap
  },
});
</script>
