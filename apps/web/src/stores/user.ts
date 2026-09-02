import { defineStore } from 'pinia'

interface UserState {
  accessToken: string | null
  refreshToken: string | null
  profile: any | null
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    accessToken: null,
    refreshToken: null,
    profile: null,
  }),
  getters: {
    isLoggedIn: (s) => !!s.accessToken,
  },
  actions: {
    setTokens(access: string, refresh: string) {
      this.accessToken = access
      this.refreshToken = refresh
      uni.setStorageSync('access_token', access)
      uni.setStorageSync('refresh_token', refresh)
    },
    setProfile(p: any) { this.profile = p },
    logout() {
      this.accessToken = null
      this.refreshToken = null
      this.profile = null
      uni.removeStorageSync('access_token')
      uni.removeStorageSync('refresh_token')
      uni.reLaunch({ url: '/pages/auth/login' })
    },
    init() {
      const a = uni.getStorageSync('access_token')
      const r = uni.getStorageSync('refresh_token')
      if (a) this.accessToken = a
      if (r) this.refreshToken = r
    },
  },
})
