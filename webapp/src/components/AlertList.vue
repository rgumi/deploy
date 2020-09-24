<template>
  <v-data-table
    :headers="headers"
    :items="alerts"
    :hide-default-footer="true"
    :disable-pagination="true"
    style="width: 100%"
  >
  </v-data-table>
</template>

<script>
export default {
  name: "AlertList",
  props: {
    alerts: Array,
  },
  data() {
    return {
      search: "",
      headers: [
        {
          text: "Type",
          sortable: true,
          value: "type",
        },
        {
          text: "Metric",
          sortable: true,
          value: "metric",
        },
        {
          text: "Threshold",
          sortable: true,
          value: "threshold",
        },
        {
          text: "Value",
          sortable: true,
          value: "value",
        },
        {
          text: "Started @",
          sortable: true,
          value: "StartTime",
        },
        {
          text: "Backend ID",
          sortable: true,
          value: "backend_id",
        },
      ],
    };
  },
};
</script>

<style></style>
