<script setup lang="ts" generic="T extends Record<string, unknown>">
import { computed, ref } from "vue";

interface MenuItem {
  label: string;
  value: T;
}

const props = defineProps<{ items: MenuItem[] }>();
const count = ref(0);
const visible = computed(() => count.value > 0);
const select = () => count.value++;
</script>

<template>
  <DropdownMenuItem
    v-if="visible"
    :data-inset="count > 0"
    @click.stop='select()'
    #default="slotProps"
    class="menu"
    disabled
  >
    <template v-if="ready"><slot /></template>
    <my-component data-test="value" />
    <section id="after-nested">{{ count + 42 }}</section>
    <p title="First line
      Second line > still inside the attribute">Multiline value</p>
    <!-- <FakeComponent> -->
    &amp;
  </DropdownMenuItem>
</template>

<style scoped>
.menu {
  color: red;
  background: #ffffff;
}
</style>
