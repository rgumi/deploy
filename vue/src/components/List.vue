<template>
  <v-container fluid>
    <v-col>
      <v-row style="margin-bottom: 20px;">
        <v-btn @click="showRunning = !showRunning">
          <v-icon
            color="rgba(0, 0, 0, 1)"
            :size="25"
            v-bind:class="{ rotate: !showRunning }"
            style="margin-right: 5px"
            >mdi-arrow-down-bold-circle-outline</v-icon
          >Running Routes
        </v-btn>
      </v-row>

      <p v-if="runningRoutes.length == 0 && showRunning">No routes found.</p>
      <div v-if="showRunning">
        <v-row v-for="(item, key) in runningRoutes" :key="key">
          <RouteComponent :route="item"></RouteComponent>
        </v-row>
      </div>
      <v-divider style="margin: 4vh"></v-divider>

      <v-row style="margin-bottom: 20px;">
        <v-btn @click="showIdle = !showIdle">
          <v-icon
            color="rgba(0, 0, 0, 1)"
            :size="25"
            v-bind:class="{ rotate: !showIdle }"
            style="margin-right: 5px"
            >mdi-arrow-down-bold-circle-outline</v-icon
          >Idle Routes
        </v-btn>
      </v-row>

      <p v-if="idleRoutes.length == 0 && showIdle">No routes found.</p>
      <div v-if="showIdle">
        <v-row v-for="item in idleRoutes" :key="item.name">
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
  props: {
    data: Object
  },
  computed: {
    runningRoutes: function() {
      console.log(this.data);
      return this.data;
    },
    idleRoutes: function() {
      console.log(this.data);
      return this.data;
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
