<template>
  <v-container fluid>
    <v-row>
      <v-col xs12 class="text-center" mt-5>
        <h1>Routes</h1>
      </v-col>
    </v-row>
    <v-row>
      <v-col xs12 class="text-center" mt-3>
        <v-progress-circular v-if="loading" :size="70" :width="7" color="grey" indeterminate></v-progress-circular>
        <Error v-if="error != null" :error="this.error"></Error>
        <List :data="this.info" v-if="this.info != null"></List>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import List from "@/components/List.vue";
import axios from "axios";
import Error from "@/components/Error.vue";

export default {
  name: "Routes",
  components: {
    List,
    Error
  },
  data: () => {
    return {
      info: null,
      loading: true,
      error: null
    };
  },

  mounted() {
    console.log("Downloading routes data");
    console.log(location.origin);
    axios
      .get(location.origin + "/v1/info")
      .then(response => {
        this.info = response.data;
        this.loading = false;
      })
      .catch(error => {
        this.error = error;
        this.loading = false;
      });
  }
};
</script>
