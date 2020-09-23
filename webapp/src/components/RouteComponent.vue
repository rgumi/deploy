<template>
  <div class="routeWrapper elevation-3">
    <!-- Info row -->
    <v-row fluid no-gutters dense>
      <div style="min-width: 15%; text-align:left">
        <h1 @click="show = !show" :disabled="currentlyLoading">{{ route.name }}</h1>
      </div>
      <v-icon
        title="Go to switchover"
        @click="$router.push(switchoverLink)"
        :disabled="currentlyLoading"
        :color="switchoverStatusColor"
        style="margin-left: 2px"
        v-if="route.switchover !== null"
      >mdi-swap-horizontal</v-icon>

      <v-icon
        title="Currently active alarms"
        :disabled="currentlyLoading"
        style="margin-left: 2px"
      >mdi-alarm-light</v-icon>

      <v-spacer></v-spacer>
      <v-icon
        title="Edit route configuration"
        @click="editRoute()"
        :disabled="currentlyLoading"
        class="routeButton editButton"
      >mdi-pencil</v-icon>
      <v-icon
        title="Remove route"
        @click="deleteRoute()"
        :disabled="currentlyLoading"
        class="routeButton delButton"
      >mdi-delete</v-icon>
      <v-icon
        title="Toggle visibility of routes"
        @click="show = !show"
        class="routeButton"
        v-bind:class="{ rotate: !show }"
        :disabled="currentlyLoading"
      >mdi-arrow-down-bold-circle</v-icon>
    </v-row>
    <v-divider></v-divider>

    <v-row fluid no-gutters dense v-if="show">
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
import store from "@/store/index";
export default {
  name: "routeComponent",
  props: {
    route: Object,
    showAll: Boolean
  },
  data() {
    return {
      show: false
    };
  },
  watch: {
    showAll() {
      this.show = this.showAll;
    }
  },
  computed: {
    routeLink: function() {
      return `/routes/${this.route.name}`;
    },
    dashboadLink: function() {
      return `/dashboard?route=${this.route.name}`;
    },
    switchoverLink: function() {
      return `/routes/${this.route.name}#switchover`;
    },
    currentlyLoading: function() {
      return store.state.loading;
    },
    switchoverStatusColor: function() {
      return "success";
    }
  },
  methods: {
    getIcon: function(status) {
      status = "running";
      let icon = this.$store.getters.getIcon(status);
      return icon;
    },
    editRoute: function() {
      alert(`Edit ${this.route.name}`);
      this.$emit("edit", this.routeName);
    },
    deleteRoute: function() {
      alert(`Delete ${this.route.name}`);
      this.$emit("delete", this.routeName);
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

.rotate {
  transform: rotate(180deg);
}
.delButton:hover {
  color: red;
}
.editButton:hover {
  color: green;
}
</style>
