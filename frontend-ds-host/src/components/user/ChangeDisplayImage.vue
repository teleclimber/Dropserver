<script setup lang="ts">
import { ref } from 'vue';
import { useAuthUserStore } from '@/stores/auth_user';
import Avatar from '../ui/Avatar.vue';

const emit = defineEmits<{
  (e: 'close'): void
}>();

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const saving = ref(false);
const error_msg = ref('');

async function avatarChanged(ev: Blob | undefined) {
	if (saving.value) return;
	saving.value = true;
	error_msg.value = '';
	try {
		if (ev) {
			await authUserStore.changeDisplayImage(ev);
		} else {
			await authUserStore.deleteDisplayImage();
		}
		emit('close');
	} catch (e) {
		error_msg.value = 'Failed to save image';
	} finally {
		saving.value = false;
	}
}
</script>

<template>
<div class="rounded border border-yellow-200 p-3 bg-yellow-100">
	<Avatar :current="authUserStore.getDisplayImageUrl()" @changed="avatarChanged"></Avatar>
	<p v-if="error_msg" class="text-red-600 mt-2">{{ error_msg }}</p>
	<p v-if="saving" class="text-gray-500 mt-2">Saving...</p>
	<div class="flex justify-start pt-2">
		<input type="button" class="btn" @click="$emit('close')" value="Cancel" />
	</div>
</div>
</template>
