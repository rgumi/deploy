<template>
    <div class="wrapper text">
        <v-row fluid no-gutters dense>
                    <table class="props" style="width:100%">
                        <tr>
                            <th>From</th>
                            <td title="from">
                            {{ switchover.from }}
                            </td>
                        </tr>
                        <tr>
                            <th>To</th>
                            <td title="to">
                            {{switchover.to }}
                            </td>
                        </tr>
                    </table>
           </v-row>
                <v-row fluid no-gutters dense>
                    <Conditions 
                        :editable="false" 
                        :conditions="switchover.conditions" 
                    />
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
    }
}
</script>

<style>

</style>