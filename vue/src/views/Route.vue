<template>
  <div>{{ this.info }}</div>
</template>

<script>
import axios from "axios";
export default {
  data: () => {
    return {
      info: null
    };
  },
  mounted() {
    let baseUrl = "http://localhost:8081";
    console.log(baseUrl);
    axios
      .get(baseUrl + "/v1/routes/" + this.$route.params.id)
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
