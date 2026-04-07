<script setup>
defineProps(['label'])
</script>
<template>
  <button
    class="flex-1 font-rs-bold uppercase md:flex-none border-2 rounded-md p-2 bg-card-red hover:bg-red-800 min-w-30 shadow-xl">
    {{ label }}
  </button>
</template>
