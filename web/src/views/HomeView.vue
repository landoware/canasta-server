<template>
  <div class="min-h-screen flex flex-col items-center justify-center text-center">
    <p class="text-card-white font-quill text-9xl leading-none text-shadow-lg mb-10">
      Canasta
    </p>
    <FormCard class="min-w-96">
      <div v-if="initialState" class="flex flex-col gap-5">
        <Button @click="router.push({ name: 'new' })" label="New Game" />
        <Button @click="joiningGame = true; initialState = false" label="Join Game" />
      </div>

      <form v-if="joiningGame" @submit.prevent="joinLobby()">
        <div class="flex flex-col gap-5">
          <input v-model.trim="roomCode" type="text" :maxlength="4"
            class="font-rs-bold text-black uppercase bg-white border border-card-blue rounded-md text-center"
            placeholder="CODE">

          <input v-model.trim="playerName" type="text"
            class="font-rs-bold text-black bg-white border border-card-blue rounded-md text-center" placeholder="Name">

          <Button type="submit" label="Join Game" />
          <Button @click="cancel()" label="Back" class="bg-card-red" />
        </div>
      </form>
    </FormCard>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import Button from '@/components/Button.vue'
import FormCard from '@/components/FormCard.vue'

const router = useRouter()

const initialState = ref(true)
const joiningGame = ref(false)

const roomCode = ref('')

function joinLobby() {
  if (!roomCode.value && !playerName.value) {
    return
  }

  if (!playerName.value) {
    return
  }

  if (!roomCode.value) {
    return
  }

  console.log('Don\'t let Amy pick up the pile')
}

function cancel() {
  initialState.value = true
  joiningGame.value = false
}

</script>
