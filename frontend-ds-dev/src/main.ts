import { createApp, reactive } from 'vue'
import App from './App.vue'

import '@/assets/css/style.css';

import twineClient from './models/twine-client';		// starts the client
twineClient.start();

import DsDevAppControl from './app-control';
export const app_control = <DsDevAppControl>reactive(new DsDevAppControl);

createApp(App).mount('#app')
