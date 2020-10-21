<template>
  <v-container fluid>
    <v-row>
      <v-col xs12 class="text-center" mt-5>
        <h1 class="avoid-clicks">
          {{ selectedRoute === null ? "Route" : selectedRoute.name }}
        </h1>
      </v-col>
    </v-row>
    <v-row justify="start">
      <!-- refresh -->
      <v-icon
        size="32"
        style="margin: 5px;"
        @click="getRoutes"
        :disabled="currentlyLoading"
        >mdi-refresh</v-icon
      >

      <!-- Open a pop up with configs for route-->
      <v-icon size="32" style="margin: 5px;" @click="sayHello">mdi-plus</v-icon>

      <v-icon
        size="32"
        style="margin: 5px;"
        v-bind:class="{ rotate: !showAll }"
        @click="showAll = !showAll"
        >mdi-arrow-down-bold-circle</v-icon
      >

      <!-- loading icon -->
      <v-progress-circular
        v-if="currentlyLoading"
        :size="24"
        :width="3"
        color="grey"
        indeterminate
        style="margin: 8px;"
      ></v-progress-circular>

      <v-spacer />
      <v-icon
        size="32"
        v-if="editable"
        title="Save changes"
        @click="saveChanges"
        :disabled="currentlyLoading"
        class="buttonIcon saveButton"
        >mdi-content-save</v-icon
      >
      <v-icon
        size="32"
        v-if="editable"
        @click="forfeitChanges"
        title="Forfeit changes"
        :disabled="currentlyLoading"
        class="buttonIcon forfeitButton"
        >mdi-close</v-icon
      >
      <v-icon
        size="32"
        v-if="!editable"
        title="Edit configuration"
        @click="editRoute()"
        :disabled="currentlyLoading"
        class="buttonIcon editButton"
        >mdi-pencil</v-icon
      >
      <v-icon
        v-if="!editable"
        size="32"
        title="Remove route"
        @click="deleteRoute()"
        :disabled="currentlyLoading"
        class="buttonIcon delButton"
        >mdi-delete</v-icon
      >
    </v-row>
    <v-row>
      <v-col xs12 class="text-center" mt-3>
        <p v-if="selectedRoute === null">No routes found.</p>
        <div v-else>
          <RouteComponent
            :editable="editable"
            :showAlerts="false"
            :showConfig="true"
            :showBackends="false"
            :showButtons="false"
            :showAll="showAll"
            :route="selectedRoute"
          ></RouteComponent>
        </div>
      </v-col>
    </v-row>

    <v-row>
      <v-col class="text-center" mt-3>
         <SwitchoverComponent 
          v-on:deleteMe="deleteSwitchover($event)"
          v-if="selectedRoute !== null && selectedRoute.switchover !== null" 
          :switchover="selectedRoute.switchover" 
          :showAll="showAll" />
      </v-col>
    </v-row>

    <v-row>
      <v-col class="text-center" mt-3>
        <p v-if="backends === []">No backend found.</p>
        <div v-else>
          <BackendComponent
            v-for="backend in backends"
            v-on:deleteBackend="deleteBackend($event)"
            :key="backend.id"
            :showAll="showAll"
            :backend="backend"
            :editable="editable"
          ></BackendComponent>
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
import RouteComponent from "@/components/RouteComponent.vue";
import BackendComponent from "@/components/BackendComponent.vue";
import SwitchoverComponent from "@/components/SwitchoverComponent";
import store from "@/store/index";
export default {
  name: "Route",
  components: {
    RouteComponent,
    BackendComponent,
    SwitchoverComponent
  },
  data: () => {
    return {
      showAll: true,
      editable: false
    };
  },
  created() {
    this.$store.commit("pullRoute");
    console.log("Routes: ", this.routes);
  },
  mounted() {},
  methods: {
    sayHello() {
      console.log("hello world");
    },
    getRoutes() {
      this.$store.commit("pullRoute");
    },
    editRoute: function() {
      console.log(`Edit ${this.selectedRoute.name}`);
      this.editable = !this.editable;
      this.$emit("edit", this.selectedRoute.name);
    },
    deleteRoute: function() {
      console.log(`Delete ${this.selectedRoute.name}`);
      this.$store.commit("deleteRoute", this.selectedRoute.name);
      this.$router.push("/routes");
    },
    forfeitChanges: function() {
      this.editable = !this.editable;
      this.$store.commit("pullRoute");
    },
    saveChanges: function() {
      this.editable = !this.editable;
      console.log(`Saving Changes on ${this.selectedRoute.name}`);
    },
    deleteBackend: function(id) {
      this.$store.commit("deleteBackend", {
        route: this.selectedRoute.name,
        backend: id
      });
    },
    deleteSwitchover: function() {
      this.$store.commit("deleteSwitchover", this.selectedRoute.name);
    }
  },
  computed: {
    configuredRoutes: function() {
      return store.state.routes;
    },
    currentlyLoading: function() {
      return store.state.loading;
    },
    selectedRoute: function() {
      var name = this.$route.params.name;
      if (name !== null) {
        var route = this.$store.getters.getRoute(name);
        return route;
      }
      return null;
    },
    backends: function() {
      if (this.selectedRoute === null) {
        return [];
      }
      return this.selectedRoute.backends;
    }
  }
};
</script>
<style scoped>
.rotate {
  transform: rotate(180deg);
}
.isEditing {
  color: green;
}
.saveButton {
  color: blue;
}
.forfeitButton {
  color: red;
}
</style>
