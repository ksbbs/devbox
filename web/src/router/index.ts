import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import Mirrors from '../views/Mirrors.vue'
import GitProxy from '../views/GitProxy.vue'
import Settings from '../views/Settings.vue'

const routes = [
  { path: '/', component: Dashboard },
  { path: '/mirrors', component: Mirrors },
  { path: '/gitproxy', component: GitProxy },
  { path: '/settings', component: Settings },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})