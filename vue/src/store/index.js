import Vue from "vue";
import Vuex from "vuex";
import { eventBus } from "@/main";
import axios from "axios";

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    iconList: {
      running: {
        color: "success",
        icon: "mdi-run",
      },
      idle: {
        color: "grey",
        icon: "mdi-run",
      },
    },
    loading: true,
    routes: [],
  },
  actions: {},
  mutations: {
    pullRoutes: function(state) {
      // let baseUrl = location.origin;
      let baseUrl = "http://192.168.0.62:9090";
      console.log(baseUrl);
      state.loading = true;
      axios
        .get(baseUrl + "/v1/info")
        .then((response) => {
          state.routes = response.data;
          state.loading = false;
        })
        .catch((error) => {
          console.error(error);
          state.loading = false;
          eventBus.$emit("showEvent", {
            icon: "mdi-alert",
            icon_color: "error",
            title: "Error",
            message: error.message,
          });
        });
    },
  },
  modules: {},
  getters: {
    getIcon: (state) => (status) => state.iconList[status],
    getRoutes: (state) => state.routes,
    getLoading: (state) => state.loading,
  },
});
