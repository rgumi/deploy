<template>
  <v-container fluid>
    <v-col>
      <p v-if="data.length == 0">No routes found.</p>
      <div>
        <v-row v-for="item in data" :key="item.name">
          <p>Hello</p>
          <RouteComponent :route="item"></RouteComponent>
        </v-row>
      </div>
    </v-col>
  </v-container>
</template>

<script>
import RouteComponent from "@/components/RouteComponent.vue";

export default {
  name: "List",
  components: { RouteComponent },
  data: () => {
    return {
      showRunning: true,
      showIdle: true
    };
  },
  created() {
    console.log("Created RouteComponent", this.data);
  },
  props: {
    data: Array
  }
};
</script>

<style scoped>
.header {
  margin-bottom: 10px;
  font-size: 4vh;
  text-align: left;
}
.rotate {
  transform: rotate(180deg);
}
</style>
