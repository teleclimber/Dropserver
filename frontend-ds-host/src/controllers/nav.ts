import {ref} from 'vue';

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

// Title

export const page_title = ref("");

export function setTitle(title :string) {
	page_title.value = title;
}
export function unsetTitle() {
	page_title.value = "";
}