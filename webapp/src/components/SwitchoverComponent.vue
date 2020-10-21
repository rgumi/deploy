<template>
   <div class="wrapper elevation-3">
    <!-- Info row -->
    <v-row fluid no-gutters dense>
      <div style="min-width: 15%; text-align:left">
        <h1 @dblclick="show = !show" >
          Switchover
        </h1>
      </div>
      <v-spacer></v-spacer>
       <v-icon
        v-if="!editable"
        size="32"
        title="Remove switchover"
        @click="deleteMe()"
        :disabled="currentlyLoading"
        class="buttonIcon delButton"
        >mdi-delete</v-icon
      >
      <v-icon
        title="Toggle visibility of switchover"
        @click="show = !show"
        class="buttonIcon"
        v-bind:class="{ rotate: !show }"
        >mdi-arrow-down-bold-circle</v-icon
      >
    </v-row>
    <v-divider></v-divider>

    <Switchover v-if="switchover != null && show" :switchover="switchover" />
   </div>
</template>

<script>
import Switchover from "@/components/Switchover";
export default {
    name: "switchoverComponent",
    components: {
        Switchover
    },
    props: {
        showAll: Boolean,
        switchover: Object,
    },
    created() {
        this.show = this.showAll;
    },
    data() {
        return {
            show: Boolean,
            editable: false,
        }
    },
    watch: {
        showAll() {
            this.show = this.showAll;
        }
    },
    computed: {
      currentlyLoading: function() {
        return this.$store.getters.getLoading;
      },
    },
    methods: {
      deleteMe: function() {
        this.$emit("deleteMe");
      }
    }
}
</script>

<style scoped>
.wrapper {
  background: white;
  min-width: 100%;
  height: auto;
  border-radius: 5px;
  margin: 5px;
}

.statusIcon {
  padding: 0;
  height: 100%;
  width: auto;
}

h1 {
  font-size: 2vh;
  padding: 1vh;
}
</style>