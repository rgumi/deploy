<template>
  <v-card v-bind:style="styleObject">
    <v-card-title> {{ route.ID }}</v-card-title>

    <v-card-text> {{ route.Value }} </v-card-text>
  </v-card>
</template>

<script>
export default {
  name: "routeComponent",
  props: {
    route: Object,
  },
  data: () => {
    return {
      styleObject: {
        "max-width": "500px",
        margin: "10px auto",
      },
    };
  },
};
</script>
