import { createApp } from 'vue'
import App from './App.vue'

import '@/assets/css/style.css';

import twineClient from './models/twine-client';		// starts the client
twineClient.start();

createApp(App).mount('#app')
