import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './assets/tailwind.css';

import twineClient from './twine-services/twine_client';

twineClient.start();

document.body.classList.add('bg-gray-100');

createApp(App).use(router).mount('body');
