<template>
  <div class="routeWrapper elevation-3">
    <!-- Info row -->
    <v-row fluid no-gutters dense>
      <h1>{{ route.name }}</h1>
      <v-spacer></v-spacer>
      <v-icon class="routeButton">mdi-pencil</v-icon>
      <v-icon class="routeButton">mdi-delete</v-icon>
    </v-row>
    <v-divider></v-divider>

    <v-row fluid no-gutters dense>
      <v-col>
        <v-row fluid class="text text-left">
          <span>
            Prefix: {{route.prefix}}
            <br />
            Methods: {{route.methods}}
            <br />
            Host: {{route.host}}
            <br />
            CookieTTL: {{route.cookie_ttl/1000000000}} seconds
            <br />
            Strategy: {{route.strategy}}
            <div v-for="backend in route.backends" :key="backend.id">
              <br />
              {{backend.name}}: {{backend.addr}}
            </div>
          </span>
        </v-row>
        <v-row fluid class="text"></v-row>
      </v-col>
      <!-- v-on:click="$router.push(dashboadLink)"
      <v-col>
        <v-icon
          size="60"
          class="statusIcon"
          :color="this.getIcon(route.Status).color"
        >{{ this.getIcon(route.Status).icon }}</v-icon>
      </v-col>
      -->
    </v-row>
    <v-divider></v-divider>
    <!-- Links-->
    <v-row fluid no-gutters dense>
      <v-btn class="ref-left" :to="routeLink">Configuration</v-btn>
      <v-btn class="ref-right" :to="dashboadLink">Dashboard</v-btn>
    </v-row>
  </div>
</template>

<script>
export default {
  name: "routeComponent",
  props: {
    route: Object
  },
  created() {
    console.log("Created RouteComponent ", this.route);
  },
  computed: {
    routeLink: function() {
      return "/routes/" + this.route.name;
    },
    dashboadLink: function() {
      return "/dashboard/" + this.route.name;
    }
  },
  methods: {
    getIcon(status) {
      status = "running";
      let icon = this.$store.getters.getIcon(status);
      return icon;
    }
  }
};
</script>

<style scoped>
.routeWrapper {
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
.text {
  padding: 15px;
  margin: 5px;
  font-weight: 500;
  font-size: 2vh;
}

.ref-right {
  width: 50%;
  border-radius: 0 0 5px 0;
}
.ref-left {
  width: 50%;
  border-radius: 0 0 0 5px;
}

.routeButton {
  margin: 5px;
}
</style>
