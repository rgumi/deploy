<template>
  <v-btn
    @click="$emit(event)"
    class="wrapper"
    align="left"
    @mouseover="hover = true"
    @mouseleave="hover = false"
    text
  >
    <v-icon left class="icon" tile :size="size">{{ btnIcon }}</v-icon>
    <span class="text hidden-xs-only" v-if="hover">{{ onHoverText }}</span>
  </v-btn>
</template>

<script>
export default {
  data() {
    return {
      hover: true,
      size: 35
    };
  },
  props: {
    onHoverText: String,
    btnIcon: String,
    btnEvent: String
  },
  computed: {
    event: function() {
      return this.btnEvent;
    }
  }
};
</script>

<style scoped>
.wrapper {
  margin: 5px;
  padding: 10px;
  position: absolute;
  z-index: 0.5;
  min-width: 150px;
}
</style>
