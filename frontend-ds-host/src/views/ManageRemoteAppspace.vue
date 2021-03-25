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
				<a :href="enter_link">{{enter_link}}</a>

				<!-- What are we going to do here?
					- change user drop id?
					- change display name and avatar (depends whether it's allowed by remote appspace? Or it'll be subject to approval)
					- Leave remote appspace (maybe "disconnect") and archive? or something...
					- Handle things that are incoming from remote, like change of owner, change of address, etc...
					- ...
				-->
			</div>
		</div>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, computed, onMounted, onUnmounted } from 'vue';
import {useRoute} from 'vue-router';

import {setTitle} from '../controllers/nav';

import {RemoteAppspace} from '../models/remote_appspaces';

import ViewWrap from '../components/ViewWrap.vue';


export default defineComponent({
	name: 'ManageRemoteAppspace',
	components: {
		ViewWrap
	},
	props: {
		domain: {
			type: String,
			required: true
		}
	},
	setup(props) {
		const route = useRoute();
		
		const remote_appspace = reactive(new RemoteAppspace);
		remote_appspace.fetch(props.domain);
		setTitle(props.domain);	// should really come from remote_appspace after it's loaded

		const enter_link = ref("http://some.link");

		onUnmounted( async () => {
			setTitle("");
		});

		return {
			remote_appspace,
			enter_link,
		}
	}
});
</script>