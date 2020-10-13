<template>
  <div>
    <v-row style="height:52px;">
      <h1>Conditions</h1>
      <v-spacer />
      <v-icon
        v-if="editable"
        size="28"
        class="addCondition"
        @click="addCondition"
        >mdi-plus</v-icon
      >
    </v-row>
    <v-row>
      <v-data-table
        :headers="headers"
        :items="conditions"
        :hide-default-footer="true"
        :disable-pagination="true"
        style="width: 100%; overflow: scroll; max-height: 210px;"
      >
        <template v-slot:item="row">
          <tr>
            <td :contenteditable="editable" title="metric" class="left">
              {{ row.item.metric }}
            </td>
            <td :contenteditable="editable" title="operator" class="left">
              {{ row.item.operator }}
            </td>
            <td :contenteditable="editable" title="threshold" class="left">
              {{ row.item.threshold }}
            </td>
            <td :contenteditable="editable" title="active_for" class="left">
              {{ row.item.active_for }}
            </td>
            <td :contenteditable="editable" title="resolve_in" class="left">
              {{ row.item.resolve_in }}
            </td>
          </tr>
        </template>
      </v-data-table>
    </v-row>
  </div>
</template>

<script>
export default {
  props: {
    editable: Boolean,
    conditions: Array
  },
  data() {
    return {
      headers: [
        {
          text: "Metric",
          sortable: true,
          value: "metric"
        },
        {
          text: "Operator",
          sortable: true,
          value: "operator"
        },
        {
          text: "Threshold",
          sortable: true,
          value: "threshold"
        },
        {
          text: "Active For",
          sortable: true,
          value: "active_for"
        }
        ,
        {
          text: "Resolve In",
          sortable: true,
          value: "resolve_in"
        }
      ]
    };
  },
  methods: {
    addCondition: function() {
      this.conditions.push({
        metric: "Metric",
        operator: "{<|>|==}",
        threshold: "Treshold",
        active_for: "Active For Duration"
      });
    }
  }
};
</script>

<style scoped>
.addCondition {
  margin: auto 10px 18px 5px;
}
</style>