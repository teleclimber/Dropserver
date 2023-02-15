import { createApp } from 'vue';
import App from './App.vue';
import { createPinia } from 'pinia';
import router from './router';
import './assets/tailwind.css';

const pinia = createPinia();

import twineClient from './twine-services/twine_client';

twineClient.start();

document.body.classList.add('bg-gray-100');

createApp(App).use(router).use(pinia).mount('body');
