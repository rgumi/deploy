<template>
  <v-container>
    <v-row>
      <v-col xs12 class="text-center">
        <h1 class="avoid-clicks">Dashboard</h1>
      </v-col>
    </v-row>
    <v-row justify="start">
      <div class="text-center">
        <v-menu offset-y @mouseleave="on=false">
          <template v-slot:activator="{ on, attrs }">
            <v-btn
              v-bind="attrs"
              v-on="on"
              style="min-width: 150px; margin-left: 10px;"
            >{{ selectedRoute }}</v-btn>
          </template>
          <v-list class="dropdown avoid-clicks" style="min-width: 150px;">
            <v-list-item v-for="(route, index) in Object.keys(data)" :key="index" ripple>
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
        <v-card class="chartContainer">
          <v-card-title>ResponseStatus</v-card-title>
          <v-card-text>
            <LineChart :options="options" :chart-data="responseStatusData"></LineChart>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" md="6">
        <v-card class="chartContainer">
          <v-card-title>TotalResponses</v-card-title>
          <v-card-text>
            <LineChart :options="options" :chart-data="totalResponseData"></LineChart>
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
import { mapState } from "vuex";
export default {
  components: {
    LineChart
  },
  data() {
    return {
      // currentlyLoading: false,
      selectedRoute: "Route",
      responseStatusData: {},
      responseTimeData: {},
      totalResponseData: {},
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
        hover: {
          // duration of animations when hovering an item
          animationDuration: 1000,
          mode: true
        },
        // animation duration after a resize
        responsiveAnimationDuration: 0,
        legend: {
          display: true
        },
        tooltips: {
          enabled: true
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
    console.log("Created dashboard");

    this.unsubscribe = this.$store.subscribe(mutation => {
      if (mutation.type === "pullMetrics") {
        console.log("Store updated");
        console.log(this.data);
        console.log(this.timestamps);
        this.fillData();
      }
    });
  },
  computed: mapState({
    data: state => state.metrics,
    timestamps: state => state.timestamps,
    currentlyLoading: state => state.loading
  }),

  methods: {
    selectRoute(routeName) {
      this.selectedRoute = routeName;
      this.fillData();
    },

    get200() {
      return this.data[this.selectedRoute].map(d => d["ResponseStatus200"]);
    },

    get300() {
      return this.data[this.selectedRoute].map(d => d["ResponseStatus300"]);
    },

    get400() {
      return this.data[this.selectedRoute].map(d => d["ResponseStatus400"]);
    },

    get500() {
      return this.data[this.selectedRoute].map(d => d["ResponseStatus500"]);
    },

    get600() {
      return this.data[this.selectedRoute].map(d => d["ResponseStatus600"]);
    },

    getResponseTime() {
      return this.data[this.selectedRoute].map(d => d["ResponseTime"]);
    },

    getTotalResponse() {
      return this.data[this.selectedRoute].map(d => d["TotalResponses"]);
    },

    fillData() {
      if (this.selectedRoute == "Route") {
        this.selectedRoute = Object.keys(this.data)[0];
      }
      if (this.data[this.selectedRoute] === undefined) {
        console.error("selectedRoute is empty");
        return;
      }

      this.responseStatusData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "2xx",
            borderColor: "rgba(50, 205, 50, 1)",
            backgroundColor: "rgba(50, 205, 50, 0.1)",
            fill: true,
            borderWidth: 1,
            data: this.get200()
          },
          {
            label: "3xx",
            borderColor: "rgba(255,216,0, 1)",
            backgroundColor: "rgba(255,216,0, 0.1)",
            fill: true,
            borderWidth: 1,
            data: this.get300()
          },
          {
            label: "4xx",
            borderColor: "rgba(0, 0, 128,1)",
            backgroundColor: "rgba(0, 0, 128,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.get400()
          },
          {
            label: "5xx",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.get500()
          },
          {
            label: "6xx",
            borderColor: "rgba(255, 0, 0, 1)",
            backgroundColor: "rgba(255, 0, 0, 0.1)",
            fill: true,
            borderWidth: 1,
            data: this.get600()
          }
        ]
      };
      this.responseTimeData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "in ms",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.getResponseTime()
          }
        ]
      };
      this.totalResponseData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.getTotalResponse()
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
