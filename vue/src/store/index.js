import Vue from "vue";
import Vuex from "vuex";
import { eventBus } from "@/main";
import axios from "axios";

Vue.use(Vuex);

export default new Vuex.Store({
  mounted() {},
  state: {
    baseUrl: "http://192.168.0.62:8081", //location.origin,
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
    usertoken: {},
    routes: {},
    metrics: {},
    timestamps: [],
    getTimestamp: function getTimestamp() {
      var d = new Date();
      var time =
        d.getHours() +
        ":" +
        (d.getMinutes() < 10 ? "0" : "") +
        d.getMinutes() +
        ":" +
        (d.getSeconds() < 10 ? "0" : "") +
        d.getSeconds();

      return time;
    },
  },
  actions: {
    async startPulling() {
      this.commit("pullMetrics");

      window.setInterval(() => {
        this.commit("pullMetrics");
      }, 5000);
    },
  },
  methods: {},
  mutations: {
    pullRoutes: function(state) {
      state.loading = true;
      axios
        .get(this.state.baseUrl + "/v1/routes/")
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
    pullMetrics: function(state) {
      state.loading = true;

      if (state.timestamps.length > 60 / 5) {
        state.timestamps.shift();
        for (var key in state.metrics) {
          state.metrics[key].shift();
        }
      }

      console.log("Pulling metrics");
      axios
        .get(
          this.state.baseUrl + "/v1/monitoring/routes?timeframe=5",
          {},
          { timeout: 2 }
        )
        .then((response) => {
          if (Object.keys(response.data).length !== 0) {
            for (var key in response.data) {
              if (key in state.metrics) {
                state.metrics[key].push(response.data[key]);
              } else {
                state.metrics[key] = new Array();
                state.metrics[key].push(response.data[key]);
              }
            }
            this.commit("incrementTimestamp");
          }
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
    incrementTimestamp: function(state) {
      state.timestamps.push(state.getTimestamp());
    },
  },
  modules: {},
  getters: {
    getIcon: (state) => (status) => state.iconList[status],
    getRoutes: (state) => state.routes,
    getLoading: (state) => state.loading,
    getMetrics: (state) => state.metrics,
    getTimestamps: (state) => state.timestamps,
  },
});
