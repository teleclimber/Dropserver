import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './assets/tailwind.css';

//document.body.classList.add('flex', 'w-full');

createApp(App).use(router).mount('body');
