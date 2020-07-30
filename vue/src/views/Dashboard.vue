<template>
  <v-container>
    <v-row>
      <v-col xs12 class="text-center" mt-5>
        <h1 class="avoid-clicks">Dashboard</h1>
      </v-col>
    </v-row>
    <v-row style="max-height:250px">
      <v-col>
        <v-card class="chartContainer">
          <v-card-title>Random Chart</v-card-title>
          <v-card-text
            ><RandomChart :options="options"></RandomChart
          ></v-card-text>
        </v-card>
      </v-col>
      <v-col>
        <v-card class="chartContainer">
          <v-card-title>Random Chart</v-card-title>
          <v-card-text
            ><RandomChart :options="options"></RandomChart
          ></v-card-text>
        </v-card>
      </v-col>

      <v-col>
        <v-card class="chartContainer">
          <v-card-title>Random Chart</v-card-title>
          <v-card-text
            ><RandomChart :options="options"></RandomChart
          ></v-card-text>
        </v-card>
      </v-col>
      <v-col>
        <v-card class="chartContainer">
          <v-card-title>Random Chart</v-card-title>
          <v-card-text
            ><RandomChart :options="options"></RandomChart
          ></v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import RandomChart from "@/components/charts/RandomChart";
export default {
  components: {
    RandomChart,
  },
  data: () => {
    return {
      options: {
        scales: {
          yAxes: [
            {
              ticks: {
                beginAtZero: true,
              },
            },
          ],
        },
        responsive: true,
        maintainAspectRatio: false,
      },
    };
  },
};
</script>

<style scoped>
.chartContainer {
}
</style>
