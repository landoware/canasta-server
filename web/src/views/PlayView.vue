<template>
  <div>
    {{ code }}
    {{ playerName }}
  </div>
</template>

<script setup>
const props = defineProps({
  code: String,
  playerName: String,
})
import { onMounted } from 'vue';
console.log(playerName)
onMounted(() => {
  const conn = new WebSocket(import.meta.env.VITE_WS_URL)

  // Send a whoami message
  // conn.send({ command: "join", roomCode: code, player: playerName })
})

</script>
