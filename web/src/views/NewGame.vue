<template>
  <div class="min-h-screen flex flex-col items-center justify-center text-center">
    <p class="text-card-white font-quill text-9xl leading-none text-shadow-lg mb-10">
      Canasta
    </p>
    <FormCard class="min-w-96">
      <form @submit.prevent="createGame()">
        <div class="flex flex-col gap-5">
          <input v-model.trim="playerName" type="text"
            class="font-rs-bold text-black bg-white border border-card-blue rounded-md text-center" placeholder="Name">

          <Button type="submit" label="Join Game" />
          <Button @click="router.push('/canasta')" label=" Back" class="bg-card-red" />
        </div>
      </form>
    </FormCard>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router';
import Button from '@/components/Button.vue';
import { ref } from 'vue';
import FormCard from '@/components/FormCard.vue';

const router = useRouter()
const playerName = ref('')
const error = ref(null)

const getCode = async () => {
  try {
    const response = await fetch(`${import.meta.env.VITE_URL}/new`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    })

    if (!response.ok) throw new Error("Failed to create game")

    const data = await response.json()

    return data.code

  } catch (err) {
    error.value = err
    return null
  }
}

async function createGame() {
  console.log('Beats dealing, doesn\'t it?')

  if (playerName.value.length > 0) {
    const code = await getCode()

    if (code) {
      router.push({ name: 'play', params: { code: code, playerName: playerName.value } })
    }
  }
}
</script>
