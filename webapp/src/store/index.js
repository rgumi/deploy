import Vue from "vue";
import Vuex from "vuex";
import { eventBus } from "@/main";
import axios from "axios";

Vue.use(Vuex);

export default new Vuex.Store({
  mounted() {
    this.startPulling();
  },
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
    routes: new Map(),
    timeframe: 60,
    granularity: 5,
    // lists all backends of route individually
    metricsIndivBackends: {},
    // backends of route are merged
    metricsSummedBackends: {},
    cumulativeData: {},

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
    merge(data) {
      return Object.values(data).reduce((a, b) => {
        for (let k in b) {
          if (k == "CustomMetrics") {
            if (a[k] === undefined) {
              a[k] = {};
            }
            for (let cm in b[k]) {
              a[k][cm] = (a[k][cm] || 0) + b[k][cm];
            }
          } else {
            a[k] = (a[k] || 0) + b[k];
          }
        }
        return a;
      }, {});
    },
  },
  actions: {
    async startPulling() {
      console.log("Initial pulling started");
      this.commit("pullRoute");
      this.commit("pullMetrics");
      this.commit("pullMetricForRoute");

      window.setInterval(() => {
        this.commit("pullMetrics");
        this.commit("pullMetricsCumulative");
      }, 5000);
    },
  },
  methods: {},
  mutations: {
    pullMetricForRoute: function(state, route) {
      state.loading = true;
      if (route === undefined) {
        route = "";
      }
      console.log("Getting metrics for /routes/" + route);
      axios
        .get(this.state.baseUrl + "/v1/monitoring/routes/" + route, null, {
          params: {
            timeframe: state.timeframe,
            granularity: state.granularity,
          },
          timeout: 2,
        })
        .then((response) => {
          console.log(response.data);

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
    pullRoute: function(state, route) {
      state.loading = true;

      if (route === undefined) {
        route = "";
      }

      axios
        .get(this.state.baseUrl + "/v1/routes/" + route, null, {
          params: {},
          timeout: 2,
        })
        .then((response) => {
          state.routes = new Array();
          if (response.data !== {}) {
            Object.keys(response.data).forEach((key) => {
              console.log(response.data[key]);
              state.routes.push(response.data[key]);
            });

            console.log("State.routes: ", state.routes);
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

    pullMetricsCumulative: function(state) {
      state.loading = true;
      axios
        .get(
          this.state.baseUrl + "/v1/monitoring/prometheus",
          {},
          { timeout: 2 }
        )
        .then((response) => {
          // if response has metrics

          if (state.cumulativeData.constructor === Object) {
            state.cumulativeData = new Map();
          }

          if (Object.keys(response.data).length > 0) {
            console.log("Object: ", state.cumulativeData);

            state.cumulativeData["timestamps"] = state.getTimestamp();
            state.cumulativeData["metrics"] = response.data;
          }
          console.log("Cumulative Data: ", state.cumulativeData);

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

      console.log("Pulling metrics");

      axios
        .get(
          this.state.baseUrl + "/v1/monitoring/backends?timeframe=5",
          {},
          { timeout: 2 }
        )
        .then((response) => {
          if (state.cumulativeData.constructor === Object) {
            state.metricsIndivBackends = new Map();

            state.metricsSummedBackends = new Map();
          }

          if (Object.keys(response.data).length > 0) {
            state.metricsIndivBackends[state.getTimestamp()] = response.data;

            console.log("metricsIndivBackends: ", state.metricsIndivBackends);
            /*
              Sum metrics by route
            */
            let tmpObject = new Map();
            Object.keys(response.data).forEach((key) => {
              tmpObject[key] = state.merge(response.data[key]);
            });

            state.metricsSummedBackends[state.getTimestamp()] = tmpObject;
          }

          console.log("metricsSummedBackends: ", state.metricsSummedBackends);

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
    getMetricsSummedBackends: (state) => state.metricsSummedBackends,
    getMetricsIndivBackends: (state) => state.metricsIndivBackends,
    getTimestamps: (state) => state.timestamps,
  },
});
