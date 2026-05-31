import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, register as registerApi, oauthMock } from '../api/auth'
import { adminLogin as adminLoginApi } from '../api/admin'

const TOKEN_KEY = 'svp_token'
const USER_KEY = 'svp_user'

function loadUser() {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const user = ref(loadUser())

  const isLoggedIn = computed(() => Boolean(token.value && user.value))
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isReviewer = computed(() => user.value?.role === 'reviewer' || user.value?.role === 'admin')

  function setSession(nextToken, nextUser) {
    token.value = nextToken
    user.value = nextUser
    localStorage.setItem(TOKEN_KEY, nextToken)
    localStorage.setItem(USER_KEY, JSON.stringify(nextUser))
  }

  async function login(form) {
    const res = await loginApi(form)
    setSession(res.data.token, res.data.user)
    return res.data
  }

  async function loginAdmin(form) {
    const res = await adminLoginApi(form)
    setSession(res.data.token, res.data.user)
    return res.data
  }

  async function register(form) {
    const res = await registerApi(form)
    return res.data
  }

  async function loginOAuthMock(oauthId = 'demo_user') {
    const res = await oauthMock({ provider: 'github', oauth_id: oauthId, nickname: 'Demo User' })
    setSession(res.data.token, res.data.user)
    return res.data
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  function updateUser(patch) {
    if (!user.value) return
    user.value = { ...user.value, ...patch }
    localStorage.setItem(USER_KEY, JSON.stringify(user.value))
  }

  return { token, user, isLoggedIn, isAdmin, isReviewer, login, loginAdmin, register, loginOAuthMock, logout, updateUser }
})
