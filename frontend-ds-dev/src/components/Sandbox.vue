<template>
<!-- sandbox feedback and some control:
  - status (running/not)
  - context: app, appspace, migrate
  - kill
  - inspect
  -->
	<div class="flex items-stretch">
		<UiButton class="mr-2 flex items-center" :class="{
				'bg-red-800':ui_inspect_sandbox,
				'border-t-red-900':ui_inspect_sandbox,
				'border-b-red-700':ui_inspect_sandbox
				}" @click.stop.prevent="toggleInspect">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" :class="{'text-cyan-800':!ui_inspect_sandbox, 'text-white': ui_inspect_sandbox}" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
			</svg>
			Inspect
		</UiButton>

		<UiButton class="flex items-center" type="submit" @click.stop.prevent="stopSandbox()">
			<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
			</svg>
			Kill
		</UiButton>
	</div>
</template>

<script lang="ts">
import { defineComponent, ref, watch, onBeforeMount } from 'vue';
import appspaceStatus from '../models/appspace-status';
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

		return {
			appspaceStatus,
			ui_inspect_sandbox,
			toggleInspect,
			stopSandbox,
		};
	}
});
</script>