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
            >{{ selectedRoute }}</v-btn>
          </template>
          <v-list class="dropdown avoid-clicks" style="min-width: 150px;">
            <v-list-item
              v-for="(route, index) in Object.keys(metricsSummedBackends)"
              :key="index"
              ripple
            >
              <v-list-item-title @click="selectRoute(route)" style="text-align:center;">{{ route }}</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>
      </div>

      <!-- refresh -->
      <!-- @click="refresh" -->
      <v-icon size="32" style="margin: 5px;" :disabled="currentlyLoading">mdi-refresh</v-icon>

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
          :data="responseStatusData"
          :title="responseStatusChartTitle"
        ></ResponseStatusChart>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" md="6">
        <v-card class="chartContainer">
          <v-card-title>Current Distribution</v-card-title>
          <v-card-text>
            <PieChart :chart-data="backendDistribution"></PieChart>
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
import { mapState } from "vuex";
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
      responseStatusData: new Map(),
      totalResponseData: {},
      responseTimeData: {},
      currentDistribution: [],
      backendDistribution: {},
      currentDistributionLabels: [],
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
  created() {
    this.unsubscribe = this.$store.subscribe(mutation => {
      if (mutation.type === "pullMetrics") {
        console.log("Store updated");

        if (this.selectedRoute == "Route") {
          this.selectedRoute = Object.keys(this.metricsSummedBackends)[0];
        }
        this.responseStatusData = this.metricsSummedBackends[
          this.selectedRoute
        ];

        console.log(this.metricsIndivBackends);

        let item =
          this.metricsIndivBackends["metrics"][this.selectedRoute].length - 1;
        this.currentDistributionLabels = Object.keys(
          this.metricsIndivBackends["metrics"][this.selectedRoute][item]
        );

        this.currentDistribution = Object.values(
          this.metricsIndivBackends["metrics"][this.selectedRoute][item]
        ).map(d => d["TotalResponses"]);

        this.fillData();
      }
    });
  },
  computed: mapState({
    metricsSummedBackends: state => state.metricsSummedBackends,
    metricsIndivBackends: state => state.metricsIndivBackends,
    currentlyLoading: state => state.loading
  }),

  methods: {
    selectRoute(routeName) {
      this.selectedRoute = routeName;
      this.responseStatusData = this.metricsSummedBackends["metrics"][
        this.selectedRoute
      ];
      this.fillData();
    },

    fillData() {
      this.responseTimeData = {
        labels: this.metricsSummedBackends.keys(),
        datasets: [
          {
            label: "in ms",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.metricsSummedBackends
              .values()
              [this.selectedRoute].map(d => d["ResponseTime"])
          }
        ]
      };

      this.backendDistribution = {
        labels: this.currentDistributionLabels,
        datasets: [
          {
            label: "",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.currentDistribution
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
