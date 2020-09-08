<template>
  <v-container fluid>
    <v-row>
      <v-col class="text-center">
        <h1>Home page</h1>
      </v-col>
    </v-row>
    <v-row>
      <v-col class="text-center" cols="12" md="8">
        <v-row class="sharedContainer">
          <v-card class="chartContainer">
            <v-card-title>Overview of my Routes</v-card-title>
            <v-card-text>
              <LineChart :options="options"></LineChart>
            </v-card-text>
          </v-card>
        </v-row>
        <v-row class="sharedContainer">
          <v-card class="chartContainer sharedContainer">
            <v-card-title>Overview of my Backends</v-card-title>
            <v-card-text>
              <LineChart :options="options"></LineChart>
            </v-card-text>
          </v-card>
        </v-row>
      </v-col>

      <!-- insert an icon with number of active alerts -->
      <v-col class="text-center" lg="4" md="4">
        <v-card class="chartContainer">
          <v-card-title>My Active Alerts</v-card-title>
          <v-card-text
            >List of Alerts with Link to Backend Dashboard</v-card-text
          >
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import LineChart from "@/components/charts/LineChart";
export default {
  name: "Home",
  components: {
    LineChart
  },
  data() {
    return {
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
  }
};
</script>

<style scoped>
.chartContainer {
  width: 100%;
}
.sharedContainer {
  margin: 0;
  margin-bottom: 12px;
}
</style>
