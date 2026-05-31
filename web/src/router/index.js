import { createRouter, createWebHistory } from 'vue-router'
import FeedView from '../views/FeedView.vue'
import HomeView from '../views/HomeView.vue'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'feed', component: FeedView, meta: { immersive: true } },
    { path: '/discover', name: 'discover', component: HomeView },
    {
      path: '/videos/:id',
      redirect: (to) => ({ path: '/', query: { v: to.params.id } })
    },
    {
      path: '/users/:id',
      name: 'user-profile',
      component: () => import('../views/UserProfileView.vue')
    },
    {
      path: '/upload',
      name: 'upload',
      component: () => import('../views/UploadView.vue')
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue')
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../views/RegisterView.vue')
    },
    {
      path: '/console/auth',
      name: 'console-auth',
      component: () => import('../views/AdminLoginView.vue'),
      meta: { hidden: true }
    },
    {
      path: '/console',
      name: 'console',
      component: () => import('../views/AdminDashboardView.vue'),
      meta: { hidden: true, requiresAdmin: true }
    },
    {
      path: '/console/review',
      name: 'console-review',
      component: () => import('../views/ReviewerView.vue'),
      meta: { hidden: true, requiresReviewer: true }
    },
    { path: '/admin/login', redirect: '/console/auth' },
    { path: '/admin', redirect: '/console' },
    { path: '/reviewer', redirect: '/console/review' }
  ],
  scrollBehavior: () => ({ top: 0 })
})

router.beforeEach((to) => {
  const auth = useAuthStore()

  if (to.meta.hidden && to.name !== 'console-auth' && !auth.isLoggedIn) {
    return { name: 'console-auth', query: { redirect: to.fullPath } }
  }

  if (to.meta.requiresAdmin && !auth.isAdmin) {
    if (auth.isReviewer) return { name: 'console-review' }
    return { name: 'console-auth', query: { redirect: to.fullPath } }
  }

  if (to.meta.requiresReviewer && !auth.isReviewer) {
    return { name: 'console-auth', query: { redirect: to.fullPath } }
  }

  return true
})

export default router
