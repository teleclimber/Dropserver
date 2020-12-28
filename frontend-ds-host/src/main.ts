import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './assets/tailwind.css';

document.body.classList.add('bg-gray-100');

createApp(App).use(router).mount('body');
