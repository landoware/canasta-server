<script setup lang="ts">
import { computed } from 'vue'
import type { ComputedRef } from 'vue'

import blueCardUrl from '@/assets/cards/blue.png'
import redCardUrl from '@/assets/cards/red.png'

interface Props {
  color?: 'red' | 'blue'
}

const props = withDefaults(defineProps<Props>(), {
  color: 'blue',
})

const backgroundStyle: ComputedRef<{ backgroundImage: string }> = computed(() => ({
  backgroundImage: `url(${props.color === 'red' ? redCardUrl : blueCardUrl})`,
}))
</script>
<template>
  <div class="relative items-center w-full overflow-hidden rounded-2xl bg-card-white max-w-96 aspect-769/1065">
    <div class="absolute -inset-1.25 bg-center bg-no-repeat bg-cover pointer-events-none" :style="backgroundStyle"
      aria-hidden="true" />

    <div class="relative z-10 h-full w-full p-4 sm:p-6 flex items-center justify-center text-center">
      <slot />
    </div>
  </div>
</template>
