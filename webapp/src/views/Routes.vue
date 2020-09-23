<template>
  <v-container fluid>
    <v-row>
      <v-col xs12 class="text-center" mt-5>
        <h1 class="avoid-clicks">Routes</h1>
      </v-col>
    </v-row>
    <v-row justify="start">
      <!-- refresh -->
      <v-icon
        size="32"
        style="margin: 5px;"
        @click="getRoutes"
        :disabled="currentlyLoading"
      >mdi-refresh</v-icon>

      <!-- Open a pop up with configs for route-->
      <v-icon size="32" style="margin: 5px;" @click="sayHello">mdi-plus</v-icon>

      <v-icon
        size="32"
        style="margin: 5px;"
        v-bind:class="{ rotate: !showAll }"
        @click="showAll= !showAll"
        :disabled="currentlyLoading"
      >mdi-arrow-down-bold-circle</v-icon>

      <!-- loading icon -->
      <v-progress-circular
        v-if="currentlyLoading"
        :size="24"
        :width="3"
        color="grey"
        indeterminate
        style="margin: 8px;"
      ></v-progress-circular>
    </v-row>
    <v-row>
      <v-col xs12 class="text-center" mt-3>
        <p v-if="Object.keys(configuredRoutes).length == 0">No routes found.</p>
        <div>
          <v-row v-for="(item) in configuredRoutes" :key="item.name">
            <RouteComponent :showAll="showAll" :route="item"></RouteComponent>
          </v-row>
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import RouteComponent from "@/components/RouteComponent.vue";
import store from "@/store/index";
export default {
  name: "Routes",
  components: {
    RouteComponent
  },
  data: () => {
    return {
      showAll: false
    };
  },
  created() {
    this.$store.commit("pullRoute");
    console.log("Routes: ", this.routes);
  },
  mounted() {},
  methods: {
    sayHello() {
      console.log("hello world");
    },
    getRoutes() {
      this.$store.commit("pullRoute");
    }
  },
  computed: {
    configuredRoutes: function() {
      return store.state.routes;
    },
    currentlyLoading: function() {
      return store.state.loading;
    }
  }
};
</script>
<style scoped>
.rotate {
  transform: rotate(180deg);
}
</style>