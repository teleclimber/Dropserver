<template>
	<div class="flex flex-col border-2 p-1 border-gray-500 rounded-lg text-sm">
		<div class="mb-1 flex justify-center items-center rounded" 
			:class="[ 	status == 2 ?	'bg-yellow-500 text-white' :
						status == 3 ?	'bg-green-600 text-white' : 
						status == 4 ?	'bg-red-500 text-white' :
										'bg-gray-200 ']">
			{{type}} sandbox {{status_str}}
		</div>
		<div class="flex items-stretch">
			<UiButton class="mr-1 flex justify-center items-center flex-grow" :pressed="ui_inspect_sandbox" @click.stop.prevent="toggleInspect">
				<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" :class="{'text-cyan-800':!ui_inspect_sandbox, 'text-white': ui_inspect_sandbox}" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
				</svg>
				Inspect
			</UiButton>
			<UiButton class="flex justify-center items-center flex-grow" type="submit" @click.stop.prevent="stopSandbox()" :disabled="!type">
				<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
				</svg>
				Kill
			</UiButton>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, ref, watch, onBeforeMount, computed } from 'vue';
import sandboxControl from '../models/sandbox-control';

import UiButton from './ui/UiButton.vue';

export default defineComponent({
	name: 'Sandbox',
	components: {
		UiButton
	},
	setup(props, context) {
		const ui_inspect_sandbox = ref(false);
		function toggleInspect() {
			ui_inspect_sandbox.value = !ui_inspect_sandbox.value;
			console.log( "setting inspect sandbox", ui_inspect_sandbox.value);
			sandboxControl.setInspect(ui_inspect_sandbox.value);
		};
		function setInspectUIToModel() {
			ui_inspect_sandbox.value = sandboxControl.inspect;
		}
		onBeforeMount( setInspectUIToModel );
		watch( () => sandboxControl.inspect, setInspectUIToModel );

		function stopSandbox() {
			sandboxControl.stopSandbox();
		}

		const type = computed( () => sandboxControl.type );
		const status = computed( () => sandboxControl.status );
		const status_str = computed( () => {
			switch(status.value) {
				case 2: return "starting";
				case 3: return "running";
				case 4: return "stopping";
				default: return "off";
			};
		})

		return {
			ui_inspect_sandbox,
			toggleInspect,
			stopSandbox,
			type, status, status_str
		};
	}
});
</script>