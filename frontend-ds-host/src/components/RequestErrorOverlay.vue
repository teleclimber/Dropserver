<template>
	<div v-if="errors.length !== 0" class="absolute top-0 left-0 w-full z-20 h-screen bg-gray-500 bg-opacity-50 p-4 flex justify-center items-center">
		<div class="w-full sm:max-w-lg rounded-md shadow-lg bg-red-100 px-4 sm:px-6 py-5">
			<h3 class="text-xl text-red-700">Request Error ({{errors.length}})</h3>
			<section class="">
				<div class="my-2 border-l-4 border-red-800 bg-white" v-for="err in errors">
					<div class="flex">
						<div class="w-16 uppercase font-bold text-gray-600 text-center">{{err.method}}</div>
						<div class="font-mono">{{err.path}}</div>
					</div>
					<div class="flex items-center">
						<div class=" w-16 font-bold text-red-800 text-lg text-center">{{err.code}}</div>
						<div class="">{{err.message}}</div>
					</div>
				</div>
			</section>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, computed } from 'vue';
import {useRoute} from 'vue-router';

import {ReqErrStack} from '../controllers/userapi';

export default defineComponent({
	name: 'RequestErrorOverlay',
	components: {
		
	},
	setup() {
		
		return {
			errors: ReqErrStack.errs,
		}
	}

});
</script>