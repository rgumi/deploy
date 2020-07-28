<template>
  <v-container fluid>
    <v-col>
      <v-row style="margin-bottom: 20px;">
        <v-btn v-on:click="showRunning = !showRunning">
          <v-icon
            color="rgba(0, 0, 0, 1)"
            :size="25"
            v-bind:class="{ rotate: !showRunning }"
            style="margin-right: 5px"
          >mdi-arrow-down-bold-circle-outline</v-icon>Running Routes
        </v-btn>
      </v-row>

      <v-row v-for="item in runningRoutes" :key="item.name">
        <RouteComponent :route="item"></RouteComponent>

        <p v-if="runningRoutes.length < 1">No routes found.</p>
      </v-row>
      <v-divider style="margin: 4vh"></v-divider>
      <v-row style="margin-bottom: 20px;">
        <v-btn v-on:click="showIdle = !showIdle">
          <v-icon
            color="rgba(0, 0, 0, 1)"
            :size="25"
            v-bind:class="{ rotate: !showIdle }"
            style="margin-right: 5px"
          >mdi-arrow-down-bold-circle-outline</v-icon>Idle Routes
        </v-btn>
      </v-row>
      <v-row v-for="item in idleRoutes" :key="item.name">
        <RouteComponent :route="item"></RouteComponent>
        <p v-if="idleRoutes.length < 1">No routes found.</p>
      </v-row>
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
  props: {
    data: Array
  },
  computed: {
    runningRoutes: function() {
      if (this.showRunning) {
        return this.data.filter(item => item.Status == "running");
      }
      return null;
    },
    idleRoutes: function() {
      if (this.showIdle) {
        return this.data.filter(item => item.Status != "running");
      }
      return null;
    }
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
