<template>
  <div class="wrapper elevation-3">
    <v-row fluid no-gutters dense>
      <div
        style="min-width: 15%; text-align:left"
        @dblclick="show = !show"
        :disabled="currentlyLoading"
      >
        <h1>Backend: {{ backend.name }}</h1>
        <span class="id-text">{{ backend.id }}</span>
      </div>
      <span
        title="Currently active alerts"
        style="margin-top: auto; margin-bottom: auto;"
      >
        <notification-bell
          class="avoid-clicks"
          :size="20"
          iconColor="rgba(0, 0, 0, 0.54)"
          :count="activeAlertsCount"
          v-if="activeAlertsCount > 0"
        />
      </span>
      <v-spacer></v-spacer>
      <v-icon
        v-if="!editable"
        size="32"
        title="Remove backend"
        @click="deleteBackend()"
        :disabled="currentlyLoading"
        class="buttonIcon delButton"
        >mdi-delete</v-icon
      >
      <v-icon
        title="Toggle visibility of backends"
        @click="show = !show"
        class="buttonIcon"
        v-bind:class="{ rotate: !show }"
        :disabled="currentlyLoading"
        >mdi-arrow-down-bold-circle</v-icon
      >
    </v-row>
    <v-divider></v-divider>

    <v-row fluid no-gutters dense v-if="show">
      <v-col>
        <v-row class="text text-left">
          <v-col>
            <table class="props" style="width:100%">
              <tr>
                <th>Address</th>
                <td :contenteditable="editable" @blur="onEdit" title="addr">
                  {{ backend.addr }}
                </td>
              </tr>
              <tr>
                <th>Weight</th>
                <td :contenteditable="editable" @blur="onEdit" title="weight">
                  {{ backend.weight }}
                </td>
              </tr>
              <tr>
                <th>Scrape URL</th>
                <td
                  :contenteditable="editable"
                  @blur="onEdit"
                  title="scrape_url"
                >
                  {{
                    backend.scrape_url == ""
                      ? "not defined"
                      : backend.scrape_url
                  }}
                </td>
              </tr>
              <tr>
                <th>Scrape Metrics</th>
                <td
                  :contenteditable="editable"
                  @blur="onEdit"
                  title="scrape_metrics"
                >
                  {{
                    backend.scrape_metrics === null
                      ? "not defined"
                      : backend.scrape_metrics
                  }}
                </td>
              </tr>
              <tr>
                <th>Healthcheck URL</th>
                <td
                  :contenteditable="editable"
                  @blur="onEdit"
                  title="healthcheck_url"
                >
                  {{
                    backend.healthcheck_url === null
                      ? "not defined"
                      : backend.healthcheck_url
                  }}
                </td>
              </tr>
            </table>
          </v-col>
          <v-col>
            <!-- Conditions -->
            <Conditions
              :editable="editable"
              :conditions="backend.metric_thresholds"
            ></Conditions>
          </v-col>
        </v-row>

        <v-divider v-if="showAlerts"></v-divider>
        <div v-if="showAlerts">
          <v-row fluid no-gutters class="text" style="padding-bottom: 0;">
            <h1>Alerts</h1>
          </v-row>
          <v-row fluid class="text">
            <AlertList :alerts="activeAlerts" />
          </v-row>
        </div>
      </v-col>
    </v-row>
  </div>
</template>

<script>
import AlertList from "@/components/AlertList";
import Conditions from "@/components/Conditions";
import NotificationBell from "vue-notification-bell";
export default {
  name: "BackendComponent",
  components: {
    NotificationBell,
    AlertList,
    Conditions
  },
  props: {
    showAll: Boolean,
    backend: Object,
    showAlerts: {
      type: Boolean,
      default: true
    },
    editable: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      show: true
    };
  },
  watch: {
    showAll() {
      this.show = this.showAll;
    }
  },
  computed: {
    activeAlerts: function() {
      return Array.from(Object.values(this.backend.active_alerts));
    },
    currentlyLoading: function() {
      return this.$store.state.loading;
    },
    activeAlertsCount: function() {
      return this.activeAlerts.length;
    }
  },
  methods: {
    edit: function() {
      alert(`Edit ${this.backend.name}`);
    },
    deleteMe: function() {
      alert(`Delete ${this.backend.name}`);
    },
    onEdit: function(e) {
      console.log(e);
      console.log(e.target.title);
      console.log(this.route.cookie_ttl);
    },
    deleteBackend: function() {
      this.$emit("deleteBackend", this.backend.id);
    }
  }
};
</script>

<style>
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
  padding-bottom: 0;
  margin-bottom: 0;
}
</style>