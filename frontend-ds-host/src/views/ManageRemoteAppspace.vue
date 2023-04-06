<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 flex flex-col sm:flex-row sm:justify-between">
				<div>
					<h3 class="text-2xl leading-6 font-medium text-gray-900">
						Manage Remote Appspace
					</h3>
				</div>
			</div>
			<div class="px-4 py-5 sm:px-6 border-t border-gray-200">
				<p class="">Remote Appspace provided by [owner dropid].</p>
				<p>{{remote_appspace.domain_name}}</p>
				<a :href="enter_link" class="text-blue-700 text-lg underline hover:text-blue-500">{{display_link}}</a>

				<!-- What are we going to do here?
					- change user drop id?
					- change display name and avatar (depends whether it's allowed by remote appspace? Or it'll be subject to approval)
					- Leave remote appspace (maybe "disconnect") and archive? or something...
					- Handle things that are incoming from remote, like change of owner, change of address, etc...
					- ...
				-->
			</div>
		</div>

		<DeleteRemoteAppspace :appspace="remote_appspace"></DeleteRemoteAppspace>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive } from 'vue';
import {useRoute} from 'vue-router';

import {RemoteAppspace} from '../models/remote_appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import DeleteRemoteAppspace from '../components/appspace/DeleteRemoteAppspace.vue';


export default defineComponent({
	name: 'ManageRemoteAppspace',
	components: {
		ViewWrap,
		DeleteRemoteAppspace
	},
	props: {
		domain: {
			type: String,
			required: true
		}
	},
	setup(props) {
		const route = useRoute();
		const display_link = ref("https:// ... loading ...");
		const enter_link = ref("/appspacelogin?appspace="+encodeURIComponent(props.domain));
		const remote_appspace = reactive(new RemoteAppspace);
		remote_appspace.fetch(props.domain).then( () => {
			const protocol = remote_appspace.no_tls ? 'http' : 'https';
			display_link.value = protocol+'://'+remote_appspace.domain_name+remote_appspace.port_string;
		});
	
		return {
			remote_appspace,
			enter_link, display_link
		}
	}
});
</script>