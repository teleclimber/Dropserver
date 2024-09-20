import { createApp } from 'vue';
import App from './App.vue';
import { createPinia } from 'pinia';
import router from './router';
import './assets/tailwind.css';

const pinia = createPinia();

document.body.classList.add('bg-gray-100');

createApp(App).use(router).use(pinia).mount('body');
