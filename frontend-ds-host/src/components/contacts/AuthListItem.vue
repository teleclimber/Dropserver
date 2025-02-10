<script setup lang="ts">
import { PostAuth } from '@/stores/appspace_users';

const props = defineProps<{
	auth: PostAuth,
	controls: boolean,
	removed: boolean
}>();

const emit = defineEmits<{
  (e: 'remove', remove: boolean): void
}>();

</script>

<template>
	<div class="flex flex-col sm:flex-row justify-between" :class="[removed ? ['bg-gray-100 ']:[]]">
		<span :class="[removed ? 'text-gray-400 line-through':[]]">
			<span v-if="auth.type=='dropid'" class="">
				DropID:
			</span>
			<span v-if="auth.type=='email'" class="">
				Email:
			</span>
			<span v-if="auth.type=='tsnetid'" class="">
				Tailscale:
			</span>
			{{ auth.identifier }}
		</span>
		<button class="btn text-red-700 text-right"
			v-if="controls && !removed"
			@click.prevent="emit('remove', true)">remove login method</button>
		<button class="btn text-right"
			v-else-if="controls"
			@click.prevent="emit('remove', false)">restore login method</button>
	</div>

</template>