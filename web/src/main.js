import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { setupHttp } from './api/http'
import './style.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
setupHttp()
app.use(router)
app.mount('#app')
