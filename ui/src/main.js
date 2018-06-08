import baseURL from './const.js'
import axios from 'axios'
import Vue from 'vue'
import BootstrapVue from "bootstrap-vue"
import VueRouter from 'vue-router'
import App from './App.vue'
import Search from './search.vue'
import Nav from './navbar.vue'
import Download from './download.vue'
import Tasks from './tasks.vue'

import "bootstrap/dist/css/bootstrap.min.css"
import "bootstrap-vue/dist/bootstrap-vue.css"

Vue.component('dna_nav', Nav)
Vue.use(BootstrapVue)
Vue.use(VueRouter)
const router = new VueRouter({
    routes: [{path: '/', component: Tasks},
        {path: '/search', name:'search', component: Search},
        {path: '/downloads', name:'downloads', component: Download},
        {path: '/tasks', name:'tasks', component: Tasks}]
})

new Vue({
  router
}).$mount('#app')
