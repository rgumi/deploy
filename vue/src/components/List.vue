<template>
  <v-container fluid>
    <div>
      <RouteComponent v-for="item in data" :key="item.name" :route="item">
      </RouteComponent>
    </div>
  </v-container>
</template>

<script>
import RouteComponent from "@/components/RouteComponent.vue";

export default {
  name: "List",
  components: { RouteComponent },
  props: {
    data: Array,
  },
};
</script>
