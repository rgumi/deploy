<template>
  <div>
    <v-alert
      type="error"
      max-width="500px"
      tile
      prominent
      light
      dismissible
      v-bind:style="styleObject"
    >
      {{ error }}
    </v-alert>
  </div>
</template>

<script>
export default {
  name: "Error",
  props: {
    error: Error,
  },
  data: () => {
    return {
      styleObject: {
        margin: "0 auto",
      },
    };
  },
};
</script>
