<template>
	<div class="py-2">
		<div class="flex justify-between">
			<h3 class="text-lg"><span class="font-medium ">{{dropid.handle}}</span> at <span class="font-medium ">{{dropid.domain_name}}</span></h3>
			<div>
				<router-link :to="{name:'dropid', query:{domain:dropid.domain_name, handle:dropid.handle}}" class="btn">Manage</router-link>
			</div>
		</div>
		<div>
			Handle: {{dropid.display_name}}
		</div>

	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {DropID} from '../models/dropids';

export default defineComponent({
	name: 'DropIDFull',
	components: {
		
	},
	props: {
		dropid: {
			type: Object as PropType<DropID>,
			required: true
		}
	}
	
});
</script>