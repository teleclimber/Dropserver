<script setup lang="ts">
import { ref, Ref, watchEffect, watch, onMounted } from 'vue';
import { useAuthUserStore} from '@/stores/auth_user';

const emit = defineEmits<{
  (e: 'close'): void
}>()

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const name_input :Ref<HTMLInputElement|null> = ref(null);
const display_name = ref("");
watchEffect( () => {
	if( authUserStore.is_loaded ) display_name.value = authUserStore.user.display_name;
});

onMounted( () => {
	if( name_input.value === null ) return;
	name_input.value.focus();
});

const save_rejected = ref('');
watch( display_name, () => {
	if( display_name.value.trim().length > 29 ) {
		save_rejected.value = 'Display name must be 29 characters or fewer';
	} else {
		save_rejected.value = '';
	}
});

const saving = ref(false);
async function saveClicked() {
	if( saving.value ) return;
	if( save_rejected.value ) return;
	saving.value = true;
	save_rejected.value = await authUserStore.changeDisplayName(display_name.value);
	saving.value = false;
	if( save_rejected.value === '' ) emit('close');
}

</script>

<template>
<div class="rounded border border-yellow-200 p-3 bg-yellow-100">
	<form @submit.prevent="saveClicked" @keyup.esc="$emit('close')">
		<input
			type="text"
			ref="name_input"
			name="display_name"
			v-model="display_name"
			class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
		<div class="bg-yellow-50 rounded px-2 mt-2">
			<p v-if="save_rejected" class="text-yellow-800 font-medium">{{ save_rejected }}</p>
			<p v-else>&nbsp;</p>
		</div>
		<div class="flex justify-between pt-2">
			<input type="button" class="btn" @click="$emit('close')" value="Cancel" />
			<input
				type="submit"
				class="btn-blue"
				:disabled="saving || !!save_rejected"
				value="Save" />
		</div>
	</form>
</div>
</template>
