<script setup lang="ts">
import { useInstanceMetaStore } from '@/stores/instance';

import ViewWrap from '../../components/ViewWrap.vue';

const instance_store = useInstanceMetaStore();
instance_store.loadData();

</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Instance information:</h3>
			</div>
			<div class="p-4 sm:px-6">
				<span class="font-medium">ds-host version:</span>&nbsp;
				<span>{{ instance_store.ds_host_version }}</span>
			</div>
		</div>
	</ViewWrap>
</template>

