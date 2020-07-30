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
        <List :data="this.routes"></List>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import List from "@/components/List.vue";

export default {
  name: "Routes",
  components: {
    List
  },
  data: () => {
    return {
      info: null,
      showRunning: true,
      showIdle: true
    };
  },
  created() {
    this.$store.commit("pullRoutes");
  },
  mounted() {
    this.info = this.$store.getters.getRoutes;
  },
  methods: {
    sayHello() {
      console.log("hello world");
    },
    getRoutes() {
      this.$store.commit("pullRoutes");
    }
  },
  computed: {
    currentlyLoading: function() {
      return this.$store.getters.getLoading;
    },
    routes: function() {
      return this.$store.getters.getRoutes;
    }
  }
};
</script>
