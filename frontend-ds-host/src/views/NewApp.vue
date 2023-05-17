<script lang="ts" setup>
import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';

const props = defineProps<{
	app_id?: number
}>();

</script>

<template>
	<ViewWrap>
		<!-- add links to existing in-process apps if any? -->
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					{{ app_id === undefined ? "Upload New App" : "Upload New Version" }}
				</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">
					Choose a Dropserver app package on your local file system.
				</p>
			</div>
			<div class="px-4 py-5 sm:px-6 ">
				<SelectFiles :app_id="props.app_id" ></SelectFiles>
			</div>
		</div>
	</ViewWrap>
</template>

