<template>
  <v-card class="chartContainer">
    <v-card-title>{{ title }}</v-card-title>
    <v-card-text>
      <LineChart :options="options" :chart-data="responseStatusData"></LineChart>
    </v-card-text>
  </v-card>
</template>

<script>
import LineChart from "@/components/charts/LineChart";
export default {
  name: "ResponseStatusChart",
  components: {
    LineChart
  },
  props: {
    statusData: Array,
    timestamps: Array,
    selected: String,
    title: String
  },
  data() {
    return {
      responseStatusData: {},
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
  watch: {
    statusData: function() {
      console.log("statusData update");
      this.fillData();
    }
  },
  methods: {
    get200() {
      return this.statusData.map(d => d["ResponseStatus200"]);
    },

    get300() {
      return this.statusData.map(d => d["ResponseStatus300"]);
    },

    get400() {
      return this.statusData.map(d => d["ResponseStatus400"]);
    },

    get500() {
      return this.statusData.map(d => d["ResponseStatus500"]);
    },

    get600() {
      return this.statusData.map(d => d["ResponseStatus600"]);
    },

    fillData() {
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
    }
  }
};
</script>