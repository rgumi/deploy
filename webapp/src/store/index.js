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
    baseUrl: location.origin, // "http://192.168.0.62:8081",
    iconList: {
      running: {
        color: "success",
        icon: "mdi-check-circle",
      },
      idle: {
        color: "grey",
        icon: "mdi-check-circle",
      },
      broken: {
        color: "red",
        icon: "mdi-check-circle",
      },
    },
    loading: true,
    routes: new Map(), // new Map()
    timeframe: 120,
    granularity: 10,
    routeMetrics: new Map(),
    backendMetrics: new Map(),
  },
  actions: {
    async startPulling() {
      this.commit("pullRoute");
      this.commit("pullMetricForBackend");
      this.commit("pullMetricForRoute");

      window.setInterval(() => {
        this.commit("pullRoute");
        this.commit("pullMetricForBackend");
        this.commit("pullMetricForRoute");
      }, this.state.granularity * 1000); // in ms
    },
    async refresh() {
      this.commit("pullRoute");
      this.commit("pullMetricForBackend");
      this.commit("pullMetricForRoute");
    },
  },
  methods: {},
  mutations: {
    pullRoute: async function(state, route) {
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
              state.routes.push(response.data[key]);
            });
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
    pullMetricForRoute: async function(state, route) {
      state.loading = true;
      if (route === undefined) {
        route = "";
      }
      axios
        .get(this.state.baseUrl + "/v1/monitoring/routes/" + route, {
          data: {},
          params: {
            timeframe: this.state.timeframe,
            granularity: this.state.granularity,
          },
          timeout: 2000,
        })
        .then((response) => {
          var myData = response.data;
          var myMap = new Map();

          Object.keys(myData).forEach((routeName) => {
            myMap.set(routeName, new Map());

            Object.keys(myData[routeName]).forEach((timestamp) => {
              var d = new Date(Date.parse(timestamp));
              var time =
                d.getHours() +
                ":" +
                (d.getMinutes() < 10 ? "0" : "") +
                d.getMinutes() +
                ":" +
                (d.getSeconds() < 10 ? "0" : "") +
                d.getSeconds();

              myMap.get(routeName).set(time, myData[routeName][timestamp]);
            });
          });
          state.routeMetrics = myMap;

          //console.log("RouteMetrics: ", state.routeMetrics);
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
    pullMetricForBackend: async function(state, backend) {
      state.loading = true;
      if (backend === undefined) {
        backend = "";
      }
      axios
        .get(this.state.baseUrl + "/v1/monitoring/backends/" + backend, {
          data: {},
          params: {
            timeframe: this.state.timeframe,
            granularity: this.state.granularity,
          },
          timeout: 2000,
        })
        .then((response) => {
          var myData = response.data;
          var myMap = new Map();

          Object.keys(myData).forEach((routeName) => {
            myMap.set(routeName, new Map());

            Object.keys(myData[routeName]).forEach((backendID) => {
              myMap.get(routeName).set(backendID, new Map());

              Object.keys(myData[routeName][backendID]).forEach((timestamp) => {
                var d = new Date(Date.parse(timestamp));
                var time =
                  d.getHours() +
                  ":" +
                  (d.getMinutes() < 10 ? "0" : "") +
                  d.getMinutes() +
                  ":" +
                  (d.getSeconds() < 10 ? "0" : "") +
                  d.getSeconds();

                myMap
                  .get(routeName)
                  .get(backendID)
                  .set(time, myData[routeName][backendID][timestamp]);
              });
            });
          });
          state.backendMetrics = myMap;
          //console.log("BackendMetrics: ", state.backendMetrics);

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
    getRoutes: (state) => {
      var routes = state.routes;
      if (routes.size == 0) {
        return new Map();
      }
      return routes;
    },
    getLoading: (state) => state.loading,
    getMetricsForBackend: (state) => {
      var metrics = state.backendMetrics;
      if (metrics.size == 0) {
        return new Map();
      }
      return metrics;
    },
    getMetricsForRoute: (state) => {
      var metrics = state.routeMetrics;
      if (metrics.size == 0) {
        return new Map();
      }
      return metrics;
    },
  },
});
