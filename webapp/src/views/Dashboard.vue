<template>
  <v-container>
    <v-row>
      <v-col xs12 class="text-center">
        <h1 class="avoid-clicks">Dashboard</h1>
      </v-col>
    </v-row>
    <v-row justify="start">
      <div class="text-center">
        <v-menu offset-y @mouseleave="on = false">
          <template v-slot:activator="{ on, attrs }">
            <v-btn
              v-bind="attrs"
              v-on="on"
              style="min-width: 150px; margin-left: 10px;"
              >{{ selectedRoute }}</v-btn
            >
          </template>
          <v-list class="dropdown avoid-clicks" style="min-width: 150px;">
            <v-list-item
              v-for="(route, index) in configuredRoutes"
              :key="index"
              ripple
            >
              <v-list-item-title
                @click="selectRoute(route)"
                style="text-align:center;"
                >{{ route }}</v-list-item-title
              >
            </v-list-item>
          </v-list>
        </v-menu>
      </div>

      <!-- refresh -->
      <!-- @click="refresh" -->
      <v-icon
        size="32"
        style="margin: 5px;"
        :disabled="currentlyLoading"
        @click="refreshDashboard"
        >mdi-refresh</v-icon
      >

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

    <!-- content -->
    <v-row>
      <v-col cols="12">
        <v-card class="chartContainer">
          <v-card-title>{{ responseStatusChartTitle }}</v-card-title>
          <v-card-text>
            <ResponseStatusChart
              :options="options"
              :data="responseData"
              :timestamps="timestamps"
            ></ResponseStatusChart>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" md="6">
        <v-card class="chartContainer">
          <v-card-title>Current Distribution</v-card-title>
          <v-card-text>
            <PieChart :chart-data="targetDistributionData"></PieChart>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="6">
        <v-card class="chartContainer">
          <v-card-title>ResponseTime</v-card-title>
          <v-card-text>
            <LineChart
              :options="options"
              :chart-data="responseTimeData"
            ></LineChart>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Backends -->
    <v-row>
      <v-col class="text-center">
        <h1 class="avoid-clicks">Backends</h1>
      </v-col>
    </v-row>
    <p class="text-center" v-if="Object.keys(backendNames).length == 0">
      No backends found.
    </p>
    <div v-else>
      <v-row>
        <div class="text-center">
          <v-btn
            v-for="(backend, index) in backendNames"
            :key="index"
            @click="selectBackend(backend)"
            v-bind:class="{ selected: isSelectedBackend(backend) }"
            style="min-width: 150px; margin-left: 5px; margin-bottom: 5px"
            >{{ backend }}</v-btn
          >
          <!--
        <v-menu offset-y @mouseleave="on = false">
          <template v-slot:activator="{ on, attrs }">
            <v-btn
              v-bind="attrs"
              v-on="on"
              style="min-width: 150px; margin-left: 10px;"
              >{{ selectedBackend }}</v-btn
            >
          </template>
          <v-list class="dropdown avoid-clicks" style="min-width: 150px;">
            <v-list-item
              v-for="(backend, index) in backendNames"
              :key="index"
              ripple
            >
              <v-list-item-title
                @click="selectBackend(backend)"
                style="text-align:center;"
                >{{ backend }}</v-list-item-title
              >
            </v-list-item>
          </v-list>
        </v-menu>
        --></div>
      </v-row>

      <BackendCharts
        :timestamps="timestamps"
        :data="backendData"
        :options="options"
        ref="backendCharts"
      ></BackendCharts>
    </div>
  </v-container>
</template>

<script>
import LineChart from "@/components/charts/LineChart";
import PieChart from "@/components/charts/PieChart";
import BackendCharts from "@/components/charts/BackendCharts";
import ResponseStatusChart from "@/components/charts/ResponseStatusChart";
import store from "@/store/index";
export default {
  components: {
    LineChart,
    ResponseStatusChart,
    PieChart,
    BackendCharts
  },
  data() {
    return {
      responseStatusChartTitle: `Response Status`,
      responseStatusData: {},
      targetDistributionData: {},
      responseTimeData: {},
      backendsMapping: new Map(),
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
          duration: 1000,
          tension: {
            duration: 1000,
            easing: "linear",
            from: 1,
            to: 0,
            loop: true
          }
        },
        tooltips: {
          mode: "x",

          intersect: false
        },
        hover: {
          mode: "index",
          intersect: false,
          axis: "x"
        },
        // animation duration after a resize
        responsiveAnimationDuration: 0,
        legend: {
          display: true
        },
        scales: {
          yAxes: [
            {
              display: true,
              ticks: {
                beginAtZero: true,
                min: 0
              },
              gridLines: {
                display: true
              }
            }
          ]
        },
        xAxes: [
          {
            gridLines: {
              display: false
            }
          }
        ]
      }
    };
  },
  computed: {
    selectedRoute: {
      get() {
        if (this.$route.query !== null) {
          var route = this.$route.query.route;
          if (route !== undefined) {
            return route;
          }
        }
        return this.configuredRoutes.length > 0
          ? this.configuredRoutes[0]
          : "Route";
      },
      set() {}
    },
    selectedBackend: {
      get() {
        if (this.$route.query !== null) {
          var backend = this.$route.query.backend;
          if (backend !== undefined) {
            return backend;
          }
        }
        return this.backendNames.length > 0 ? this.backendNames[0] : "Backend";
      },
      set() {}
    },
    configuredRoutes: function() {
      var keys = new Array();
      Object.keys(store.state.routes).forEach(key => {
        keys.push(store.state.routes[key].name);
      });

      return keys;
    },
    routeMetrics: function() {
      return this.$store.getters.getMetricsForRoute;
    },
    backendMetrics: function() {
      return this.$store.getters.getMetricsForBackend;
    },
    currentlyLoading: function() {
      return this.$store.getters.getLoading;
    },
    responseData: function() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.routeMetrics.size == 0
      ) {
        return [];
      }
      return Array.from(this.routeMetrics.get(this.selectedRoute).values());
    },
    timestamps: function() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.routeMetrics.size == 0
      ) {
        return [];
      }

      return Array.from(this.routeMetrics.get(this.selectedRoute).keys());
    },
    targetDistribution() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.backendMetrics.size == 0
      ) {
        return [];
      }
      var backendValues = this.backendMetrics.get(this.selectedRoute);
      var values = new Array();

      backendValues.forEach(val => {
        values.push(
          Array.from(val.values())
            .map(d => d["TotalResponses"])
            .reduce((sum, entry) => sum + entry)
        );
      });
      return values;
    },
    responseTimes() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.routeMetrics.size == 0
      ) {
        return [];
      }
      return Array.from(this.routeMetrics.get(this.selectedRoute).values()).map(
        d => d["ResponseTime"]
      );
    },
    backendOfSelectedRoute() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.backendMetrics.size == 0
      ) {
        return [];
      }
      var backendValues = this.backendMetrics.get(this.selectedRoute);
      return Array.from(backendValues.keys());
    },
    backendData() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.backendMetrics.size == 0
      ) {
        return [];
      }
      var id = this.backendsMapping.get(this.selectedBackend);
      console.log(id);
      if (id == undefined || this.backendMetrics === null) {
        console.log("failed at 1");
        return [];
      }
      var metricsOfRoute = this.backendMetrics.get(this.selectedRoute);
      if (metricsOfRoute === null) {
        console.log("failed at 2");
        return [];
      }
      var metricsOfBackend = metricsOfRoute.get(id);
      if (metricsOfBackend === undefined) {
        console.log("failed at 3");
        return [];
      }
      return Array.from(metricsOfBackend.values());
    },
    backendNames() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.backendMetrics.size == 0
      ) {
        return [];
      }
      var backendNames = [];
      Array.from(this.backendMetrics.get(this.selectedRoute).keys()).forEach(
        id => {
          var name = this.$store.getters.getNameOfBackend(
            this.selectedRoute,
            id
          );
          backendNames.push(name);
          this.backendsMapping.set(name, id);
        }
      );
      return backendNames;
    }
  },

  watch: {
    selectedRoute: function() {
      this.fillData();
    },
    routeMetrics: function() {
      this.fillData();
    },
    backendMetrics: function() {
      console.log(`Backend Metrics updated`);
      this.fillData();
      this.$refs.backendCharts.fillData();
    }
  },
  methods: {
    selectRoute(routeName) {
      if (this.selectedRoute == routeName) {
        return;
      }
      this.selectedRoute = routeName;
      this.$router.push({
        query: { route: routeName, backend: this.selectedBackend }
      });
    },
    selectBackend(backendName) {
      if (this.selectedBackend == backendName) {
        return;
      }
      this.selectedBackend = backendName;
      this.$router.push({
        query: { route: this.selectedRoute, backend: backendName }
      });
    },
    refreshDashboard() {
      store.dispatch("refresh");
    },
    isSelectedBackend(backend) {
      return backend == this.selectedBackend ? true : false;
    },

    fillData() {
      if (
        this.selectedRoute == "Route" ||
        this.selectedRoute === undefined ||
        this.routeMetrics.size == 0
      ) {
        return;
      }
      this.responseTimeData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "in ms",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.responseTimes
          }
        ]
      };

      this.targetDistributionData = {
        labels: this.backendNames,
        datasets: [
          {
            label: "",
            borderColor: [
              "rgba(50, 205, 50, 1)",
              "rgba(0, 0, 128,1)",
              "rgba(255,216,0, 1)"
            ],
            backgroundColor: [
              "rgba(50, 205, 50, 0.3)",
              "rgba(0, 0, 128,0.3)",
              "rgba(255,216,0, 0.3)"
            ],
            fill: true,
            borderWidth: 1,
            data: this.targetDistribution
          }
        ]
      };
    }
  }
};
</script>

<style scoped>
.chartContainer {
  width: 100%;
}
.dropdown {
  cursor: pointer;
}
.selected {
  background-color: rgba(0, 0, 0, 0.3) !important;
}
</style>
