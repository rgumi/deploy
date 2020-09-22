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
              href="#Routes"
              v-bind="attrs"
              v-on="on"
              style="min-width: 150px; margin-left: 10px;"
            >{{ selectedRoute }}</v-btn>
          </template>
          <v-list class="dropdown avoid-clicks" style="min-width: 150px;">
            <v-list-item v-for="(route, index) in configuredRoutes" :key="index" ripple>
              <v-list-item-title @click="selectRoute(route)" style="text-align:center;">{{ route }}</v-list-item-title>
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
      >mdi-refresh</v-icon>

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
        <ResponseStatusChart
          class="chartContainer"
          :options="options"
          :data="responseData"
          :timestamps="timestamps"
          :title="responseStatusChartTitle"
        ></ResponseStatusChart>
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
            <LineChart :options="options" :chart-data="responseTimeData"></LineChart>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import LineChart from "@/components/charts/LineChart";
import PieChart from "@/components/charts/PieChart";
import ResponseStatusChart from "@/components/charts/ResponseStatusChart";
import store from "@/store/index";
// import { mapState } from "vuex";
export default {
  components: {
    LineChart,
    ResponseStatusChart,
    PieChart
  },
  data() {
    return {
      // currentlyLoading: false,
      selectedRoute: "Route",
      responseStatusChartTitle: "Response Status of Route",
      responseStatusData: {},
      targetDistributionData: {},
      responseTimeData: {},
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
    configuredRoutes: function() {
      var keys = new Array();
      Object.keys(store.state.routes).forEach(key => {
        keys.push(store.state.routes[key].name);
      });

      return keys;
    },
    routeMetrics: function() {
      return store.state.routeMetrics;
    },
    backendMetrics: function() {
      return store.state.backendMetrics;
    },
    currentlyLoading: function() {
      return store.state.loading;
    },
    responseData: function() {
      if (this.selectedRoute == "Route" || this.selectedRoute === undefined) {
        return [];
      }

      return Array.from(
        store.state.routeMetrics.get(this.selectedRoute).values()
      );
    },
    timestamps: function() {
      if (this.selectedRoute == "Route" || this.selectedRoute === undefined) {
        return [];
      }

      return Array.from(
        store.state.routeMetrics.get(this.selectedRoute).keys()
      );
    },
    targetDistribution() {
      if (this.selectedRoute == "Route" || this.selectedRoute === undefined) {
        return [];
      }
      var backendValues = store.state.backendMetrics.get(this.selectedRoute);
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
    targetsName() {
      if (this.selectedRoute == "Route" || this.selectedRoute === undefined) {
        return [];
      }

      return Array.from(
        store.state.backendMetrics.get(this.selectedRoute).keys()
      );
    },
    responseTimes() {
      if (this.selectedRoute == "Route" || this.selectedRoute === undefined) {
        return [];
      }

      return Array.from(
        store.state.routeMetrics.get(this.selectedRoute).values()
      ).map(d => d["ResponseTime"]);
    }
  },

  watch: {
    routeMetrics: function() {
      if (this.selectedRoute == "Route" || this.selectedRoute === undefined) {
        if (this.configuredRoutes.length > 0) {
          this.selectRoute(this.configuredRoutes[0]);
        }
      }
      this.fillData();
    },
    backendMetrics: function() {
      this.fillData();
    }
  },
  methods: {
    selectRoute(routeName) {
      this.selectedRoute = routeName;
      this.fillData();
    },
    refreshDashboard() {
      store.dispatch("refresh");
    },
    fillData() {
      if (this.selectedRoute == "Route" || this.selectedRoute === undefined) {
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
        labels: this.targetsName,
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
</style>
