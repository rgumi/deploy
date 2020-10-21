<template>
    <div style="width:100%">
        <v-row fluid no-gutters dense>
            <v-col>
                <table class="props text" style="width:100%">
                    <tr>
                        <th>Status</th>
                        <td title="status">
                        {{ switchover.status }}
                        </td>
                    </tr>
                    <tr>
                        <th>From -> To</th>
                        <td title="from">
                        {{ switchover.from }} -> {{switchover.to }}
                        </td>
                    </tr>
                    <tr>
                        <th>Timeout</th>
                        <td title="timeout">
                        {{switchover.timeout }}
                        </td>
                    </tr>
                    <tr>
                        <th>Weight Change</th>
                        <td title="weight_change">
                        {{switchover.weight_change }}
                        </td>
                    </tr>
                    <tr>
                        <th>Rollback</th>
                        <td title="rollback">
                        {{switchover.rollback }}
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <v-row style="height:52px;" v-if="editable">
            <v-spacer />
            <v-icon
                size="28"
                class="addCondition"
                @click="addCondition"
                >mdi-plus</v-icon>
        </v-row>
        <v-row fluid no-gutters dense class="text">
            <Conditions 
                :editable="false" 
                :conditions="switchover.conditions" />
        </v-row>
    </div>
</template>

<script>
import Conditions from "@/components/Conditions";
export default {
    name: "Switchover",
    props: {
        switchover: Object
    },
    components: {
        Conditions
    },
    data() {
    return {
        editable: false,

      headers: [
        {
          text: "Status",
          sortable: false,
          value: "status"
        },
        {
          text: "From Backend",
          sortable: false,
          value: "from"
        },
        {
          text: "To Backend",
          sortable: false,
          value: "to"
        },
      ]};
    },
}
</script>

<style>

</style>