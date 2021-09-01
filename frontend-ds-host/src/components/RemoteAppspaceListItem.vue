<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 flex flex-col sm:flex-row sm:justify-between">
			<div>
				<h3 class="text-2xl leading-6 font-medium text-gray-900">
					{{remote_appspace.domain_name}}
				</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">Remote Appspace provided by [owner dropid].</p>
			</div>
		</div>
		<div class="px-4 py-5 sm:px-6 border-t border-gray-200">
			<a :href="enter_link" class="text-blue-700 text-lg underline hover:text-blue-500">{{display_link}}</a>
		</div>
		<div class="px-4 py-5 sm:px-6 flex justify-end border-t border-gray-200">
			<router-link :to="{name: 'manage-remote-appspace', params:{domain:remote_appspace.domain_name}}" class="btn btn-blue">Manage</router-link>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType, ref } from 'vue';

import type {RemoteAppspace} from '../models/remote_appspaces';

export default defineComponent({
	name: 'RemoteAppspaceListItem',
	components: {
	},
	props: {
		remote_appspace: {
			type: Object as PropType<RemoteAppspace>,
			required: true
		}
	},
	setup(props) {
		const protocol = props.remote_appspace.no_tls ? 'http' : 'https';
		const display_link = ref(protocol+'://'+props.remote_appspace.domain_name+props.remote_appspace.port_string);

		const enter_link = ref("/appspacelogin?appspace="+encodeURIComponent(props.remote_appspace.domain_name));

		return {
			display_link,
			enter_link,
		}
	}
	
});
</script>