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
    baseUrl: "http://192.168.0.62:8081", // location.origin, // ,
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
    routes: new Map(),
    routeMetrics: new Map(),
    backendMetrics: new Map(),
    timeframe: 120,
    granularity: 10,
    editing: false,
  },
  actions: {
    async startPulling() {
      this.commit("pullRoute");
      this.commit("pullMetricForBackend");
      this.commit("pullMetricForRoute");

      window.setInterval(() => {
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
      if (state.editing) {
        return;
      }
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
    deleteRoute: async function(state, routeName) {
      if (state.editing) {
        return;
      }
      state.loading = true;
      console.log(routeName);
      axios
        .delete(this.state.baseUrl + "/v1/routes/" + routeName)
        .then((response) => {
          if (response.status == 200) {
            // route successfully deleted
            this.commit("pullRoute");
          }
          state.loading = false;
        })
        .catch((error) => {
          state.loading = false;
          console.error(error);
          eventBus.$emit("showEvent", {
            icon: "mdi-alert",
            icon_color: "error",
            title: "Error",
            message: error.message,
          });
        });
    },
    setEditing: async function(state, status) {
      state.editing = status;
    },
  },
  modules: {},
  getters: {
    getIcon: (state) => (status) => state.iconList[status],
    getActiveAlerts: (state) => (routeName) => {
      console.log(`Requested ${routeName}`);

      var routes = state.routes;
      if (routes.size == 0) {
        return [];
      }
      var route = routes.find((element) => element.name === routeName);
      if (route === undefined) {
        return [];
      }
      var activeAlerts = Array.from(
        Object.entries(route.backends)
          .map((d) => d[1])
          .map((d) => d.active_alerts)
          .filter((d) => Object.keys(d).length !== 0)
      ).map((d) => Object.values(d)[0]);

      return activeAlerts;
    },
    getRoutes: (state) => {
      var routes = state.routes;
      if (routes.size == 0) {
        return new Map();
      }
      return routes;
    },
    getRoute: (state) => (routeName) => {
      // console.log(`Requested ${routeName}`);

      var routes = state.routes;
      if (routes.size == 0) {
        return null;
      }
      var route = routes.find((element) => element.name === routeName);
      if (route === undefined) {
        return null;
      }
      return route;
    },
    getLoading: (state) => state.loading,
    getEditing: (state) => state.editing,
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
