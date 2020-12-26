import {ref} from 'vue';

console.log("running nav module");

export const nav_open = ref(false);

export function openNav() {
	console.log("setting nav open");
	nav_open.value = true;
}

export function closeNav() {
	nav_open.value = false;
}