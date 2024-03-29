import {ref} from 'vue';

// this could really be a local pinia store?
// at least for "nav_open"

export const nav_open = ref(false);

export function openNav() {
	nav_open.value = true;
}

export function closeNav() {
	nav_open.value = false;
}

// User dropdown menu:

export const user_menu_open = ref(false);

export function openUserMenu() {
	user_menu_open.value = true;
}

export function closeUserMenu() {
	user_menu_open.value = false;
}
