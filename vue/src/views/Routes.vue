<template>
  <v-container fluid>
    <v-row>
      <v-col xs12 class="text-center" mt-5>
        <h1>Routes</h1>
      </v-col>
    </v-row>
    <v-row justify="start">
      <!-- refresh -->
      <v-icon size="32" style="margin: 5px;" @click="getRoutes">mdi-refresh</v-icon>

      <!-- Open a pop up with configs for route-->
      <v-icon size="32" style="margin: 5px;" @click="sayHello">mdi-plus</v-icon>
    </v-row>
    <v-row>
      <v-col xs12 class="text-center" mt-3>
        <v-progress-circular v-if="loading" :size="70" :width="7" color="grey" indeterminate></v-progress-circular>
        <List :data="this.info" v-if="this.info != null"></List>
        <p v-if="!loading&this.info == null">Could not find any routes.</p>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import List from "@/components/List.vue";
import axios from "axios";
import { eventBus } from "@/main";

export default {
  name: "Routes",
  components: {
    List
  },
  data: () => {
    return {
      info: null,
      loading: true,
      error: null
    };
  },

  mounted() {
    this.getRoutes();
  },
  methods: {
    sayHello() {
      console.log("hello world");
    },
    getRoutes() {
      // let baseUrl = location.origin;
      let baseUrl = "http://localhost:9090";
      console.log(baseUrl);
      this.loading = true;
      this.info = null;
      axios
        .get(baseUrl + "/v1/info")
        .then(response => {
          this.info = response.data;
          this.loading = false;
        })
        .catch(error => {
          console.error(error);
          this.loading = false;
          eventBus.$emit("showEvent", {
            icon: "mdi-alert",
            icon_color: "error",
            title: "Error",
            message: error.message
          });
        });
    }
  }
};
</script>
