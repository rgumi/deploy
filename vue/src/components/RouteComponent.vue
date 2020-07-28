<template >
  <div class="routeWrapper">
    <!-- Info row -->
    <v-row fluid no-gutters dense>
      <h1>{{route.Value}} ({{ route.ID }})</h1>
      <v-spacer></v-spacer>
      <v-icon class="routeButton">mdi-pencil</v-icon>
      <v-icon class="routeButton">mdi-delete</v-icon>
    </v-row>
    <v-divider></v-divider>

    <v-row fluid no-gutters dense>
      <!-- v-on:click="$router.push(routeLink)" -->
      <v-col>
        <v-row fluid class="text">
          <span>
            {{route.From}}
            &#8594;
            {{route.To}}
          </span>
        </v-row>
      </v-col>
      <!-- v-on:click="$router.push(dashboadLink)"-->
      <v-col>
        <v-icon
          size="60"
          class="statusIcon"
          :color="this.getIcon(route.Status).color"
        >{{ this.getIcon(route.Status).icon }}</v-icon>
      </v-col>
    </v-row>
    <v-divider></v-divider>
    <!-- Links-->
    <v-row fluid no-gutters dense>
      <v-btn class="reference" :to="routeLink">Configuration</v-btn>
      <v-btn class="reference" :to="dashboadLink">Dashboard</v-btn>
    </v-row>
  </div>
</template>

<script>
export default {
  name: "routeComponent",
  props: {
    route: Object
  },
  computed: {
    routeLink: function() {
      return "/routes/" + this.route.ID;
    },
    dashboadLink: function() {
      return "/dashboard/" + this.route.ID;
    }
  },
  methods: {
    getIcon(status) {
      let icon = this.$store.getters.getIcon(status);
      return icon;
    }
  }
};
</script>


<style scoped>
.routeWrapper {
  min-width: 100%;
  max-width: 100%;
  height: auto;
  border: 2px;
  border-color: rgba(139, 136, 136, 0.87);
  border-style: solid;
  margin-bottom: 5px;
  margin-top: 5px;
  margin-right: 5px;
  margin-left: 5px;
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
.text {
  padding: 5%;
  margin: 5px;
  font-weight: 500;
  font-size: 2vh;
}

.reference {
  width: 50%;
}

.routeButton {
  margin: 5px;
}
</style>