<template>
  <div class="small">
    <LineChart :chart-data="datacollection"></LineChart>
    <v-btn @click="fillData()">Randomize</v-btn>
  </div>
</template>

<script>
import LineChart from "./LineChart.js";
//import PieChart from "./PieChart.js";
export default {
  components: {
    LineChart,
  },
  data() {
    return {
      datacollection: null,
    };
  },
  mounted() {
    this.fillData();
  },
  methods: {
    fillData() {
      this.datacollection = {
        labels: [this.getRandomInt(), this.getRandomInt(), this.getRandomInt()],
        datasets: [
          {
            label: "Data One",
            backgroundColor: "#f87979",
            data: [
              this.getRandomInt(),
              this.getRandomInt(),
              this.getRandomInt(),
            ],
          },
          {
            label: "Data Two",
            backgroundColor: "#f89456",
            data: [
              this.getRandomInt(),
              this.getRandomInt(),
              this.getRandomInt(),
            ],
          },
        ],
      };
    },
    getRandomInt() {
      return Math.floor(Math.random() * (50 - 5 + 1)) + 5;
    },
  },
};
</script>

<style>
.small {
  margin: auto;
  position: relative;
  height: 100%;
}
</style>
