import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import Mirrors from '../views/Mirrors.vue'
import GitProxy from '../views/GitProxy.vue'
import Settings from '../views/Settings.vue'
import Login from '../views/Login.vue'
import Search from '../views/Search.vue'
import { isLoggedIn, checkAuthRequired, initAuth } from '../api/client'

const routes = [
  { path: '/login', component: Login, meta: { noAuth: true } },
  { path: '/', component: Dashboard },
  { path: '/mirrors', component: Mirrors },
  { path: '/gitproxy', component: GitProxy },
  { path: '/search', component: Search },
  { path: '/settings', component: Settings },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

let authChecked = false

router.beforeEach(async (to) => {
  if (!authChecked) {
    initAuth()
    authChecked = true
  }

  if (to.meta.noAuth) return true

  if (isLoggedIn()) return true

  const required = await checkAuthRequired()
  if (!required) return true

  return '/login'
})

export default router