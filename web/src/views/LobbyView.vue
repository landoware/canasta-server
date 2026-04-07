<script setup>
defineProps({
  code: String,
  playerName: String,
})

import { useRouter } from 'vue-router';
import Button from '@/components/Button.vue';
import LobbyPlayerCard from '@/components/LobbyPlayerCard.vue';
import FormCard from '@/components/FormCard.vue';


const router = useRouter()


function ready() {
}

function leave() {
  router.push('/canasta')
}

</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center text-center">

    <div v-if="hasName">
      <div class="text-card-white text-[clamp(2.5rem,14vw,6rem)] text-shadow-lg">
        <span class="font-quill">Room </span>
        <span class="font-rs-bold text-rs-yellow">{{ id }}</span>
      </div>
      <div class="text-card-white font-quill text-[clamp(2.5rem,14vw,6rem)] text-shadow-lg">
        Pick your Partner
      </div>
      <div class="grid grid-cols-2 justify-center items-center gap-10">
        <div>
          <!-- <LobbyPlayerCard :playerName="gameStore.lobbyState?.players[0]?.username" -->
          <!--   :ready="gameStore.lobbyState?.players[0]?.ready" :team="true" /> -->
        </div>
        <div>
          <!-- <LobbyPlayerCard :playerName="gameStore.lobbyState?.players[1]?.username" -->
          <!--   :ready="gameStore.lobbyState?.players[1]?.ready" /> -->
        </div>
        <div>
          <!-- <LobbyPlayerCard :playerName="gameStore.lobbyState?.players[2]?.username" -->
          <!--   :ready="gameStore.lobbyState?.players[2]?.ready" :team="true" /> -->
        </div>
        <div>
          <!-- <LobbyPlayerCard :playerName="gameStore.lobbyState?.players[3]?.username" -->
          <!--   :ready="gameStore.lobbyState?.players[3]?.ready" :team="false" /> -->
        </div>
      </div>
      <Button @click="ready()" label="Ready" class="m-5" />
      <Button @click="leave()" label="Leave" />

    </div>
  </div>
</template>
