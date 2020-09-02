<template>
  <v-container>
    <v-row>
      <v-col xs12 class="text-center" mt-5>
        <h1 class="avoid-clicks">Dashboard</h1>
      </v-col>
    </v-row>
    <v-row>
      <v-col>
        <v-card class="chartContainer">
          <v-card-title>ResponseStatus</v-card-title>
          <v-card-text>
            <LineChart :options="options" :chart-data="responseStatusData"></LineChart>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col>
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
import axios from "axios";
import LineChart from "@/components/charts/LineChart";
export default {
  components: {
    LineChart
  },
  data() {
    return {
      responseStatusData: Object,
      responseTimeData: Object,
      data: Array,
      timestamps: Array,
      options: {
        responsive: true,
        maintainAspectRatio: false,

        animation: {
          duration: 0
        },
        legend: {
          display: true
        },
        tooltips: {
          enabled: true
        },
        hover: {
          mode: true
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
                display: true // my new default options
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

  mounted: function() {
    this.data = [];
    this.timestamps = [];

    this.fillData();
    this.getData();

    this.$nextTick(function() {
      window.setInterval(() => {
        if (this.timestamps.length > 60 / 5) {
          this.timestamps.shift();
          this.data.shift();
        }
        this.getData();
      }, 5000);
    });
  },

  methods: {
    getData() {
      let baseUrl = "http://localhost:8081";
      axios
        .get(baseUrl + "/v1/monitoring/")
        .then(response => {
          this.data.push(response.data);
          this.timestamps.push(this.getTimestamp());
          this.fillData();
          this.loading = false;
        })
        .catch(error => {
          this.error = error;
          console.error(error);
          this.loading = false;
        });
    },

    get200() {
      return this.data.map(d => d["TestRoute"]["ResponseStatus200"]);
    },

    get600() {
      return this.data.map(d => d["TestRoute"]["ResponseStatus600"]);
    },

    getResponseTime() {
      return this.data.map(d => d["TestRoute"]["ResponseTime"]);
    },

    getTimestamp() {
      var d = new Date();
      var time = d.getHours() + ":" + d.getMinutes() + ":" + d.getSeconds();
      return time;
    },

    fillData() {
      this.responseStatusData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "2xx",
            borderColor: "#32CD32",
            fill: false,
            borderWidth: 1,
            data: this.get200()
          },
          {
            label: "6xx",
            borderColor: "#FF0000",
            fill: false,
            borderWidth: 1,
            data: this.get600()
          }
        ]
      };
      this.responseTimeData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "2xx",
            borderColor: "#282828",
            fill: false,
            borderWidth: 1,
            data: this.getResponseTime()
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
</style>
