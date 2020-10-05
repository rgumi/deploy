<template>
  <div>
    <v-row>
      <v-col cols="12">
        <v-card class="chartContainer">
          <v-card-title>Response Status</v-card-title>
          <v-card-text>
            <ResponseStatusChart
              :options="options"
              :data="responseStatus"
              :timestamps="timestamps"
            ></ResponseStatusChart>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" md="6">
        <v-card class="chartContainer">
          <v-card-title>Response Time</v-card-title>
          <v-card-text>
            <LineChart
              :options="options"
              :chart-data="responseTimeData"
            ></LineChart>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="6">
        <v-card class="chartContainer">
          <v-card-title>Content Length</v-card-title>
          <v-card-text>
            <LineChart
              :options="options"
              :chart-data="contentLengthData"
            ></LineChart>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script>
import ResponseStatusChart from "@/components/charts/ResponseStatusChart";
import LineChart from "@/components/charts/LineChart";
export default {
  name: "BackendCharts",
  components: {
    ResponseStatusChart,
    LineChart
  },
  props: {
    data: Array,
    timestamps: Array,
    options: {}
  },
  data() {
    return {
      responseTimeData: {},
      contentLengthData: {}
    };
  },
  computed: {
    responseStatus() {
      if (this.data.length > 0) {
        return this.data;
      }
      return [];
    },
    responseTime() {
      if (this.data.length > 0) {
        return this.data.map(d => d["ResponseTime"]);
      }
      return [];
    },
    contentLength() {
      if (this.data.length > 0) {
        return this.data.map(d => d["ContentLength"]);
      }
      return [];
    }
  },
  watch: {
    data() {
      console.log(this.timestamps);
      this.fillData();
    }
  },
  methods: {
    fillData() {
      console.log("Updating BackendCharts");
      this.responseTimeData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "in ms",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.responseTime
          }
        ]
      };

      this.contentLengthData = {
        labels: this.timestamps,
        datasets: [
          {
            label: "in bytes",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.contentLength
          }
        ]
      };
    }
  }
};
</script>