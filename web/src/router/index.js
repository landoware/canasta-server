import HomeView from '@/views/HomeView.vue'
import LobbyView from '@/views/LobbyView.vue'
import NewGame from '@/views/NewGame.vue'
import PlayView from '@/views/PlayView.vue'
import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/canasta', component: HomeView, name: 'home' },
    { path: '/canasta/new-game', component: NewGame, name: 'new' },
    { path: '/canasta/:code', component: PlayView, name: 'play', props: route => ({ code: route.params.code, playerName: route.params.playerName }) }
  ],
})

export default router
