<template>
  <LineChart :options="options" :chart-data="responseStatusData"></LineChart>
</template>

<script>
import LineChart from "@/components/charts/LineChart";
export default {
  name: "ResponseStatusChart",
  components: {
    LineChart
  },
  props: {
    data: Array,
    timestamps: Array,
    options: {}
  },
  data() {
    return {
      responseStatusData: {}
    };
  },
  watch: {
    data: function() {
      this.fillData();
    }
  },
  methods: {
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
            data: this.data.map(d => d["ResponseStatus200"])
          },
          {
            label: "3xx",
            borderColor: "rgba(255,216,0, 1)",
            backgroundColor: "rgba(255,216,0, 0.1)",
            fill: true,
            borderWidth: 1,
            data: this.data.map(d => d["ResponseStatus300"])
          },
          {
            label: "4xx",
            borderColor: "rgba(0, 0, 128,1)",
            backgroundColor: "rgba(0, 0, 128,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.data.map(d => d["ResponseStatus400"])
          },
          {
            label: "5xx",
            borderColor: "rgba(96,96,96,1)",
            backgroundColor: "rgba(96,96,96,0.1)",
            fill: true,
            borderWidth: 1,
            data: this.data.map(d => d["ResponseStatus500"])
          },
          {
            label: "6xx",
            borderColor: "rgba(255, 0, 0, 1)",
            backgroundColor: "rgba(255, 0, 0, 0.1)",
            fill: true,
            borderWidth: 1,
            data: this.data.map(d => d["ResponseStatus600"])
          }
        ]
      };
    }
  }
};
</script>