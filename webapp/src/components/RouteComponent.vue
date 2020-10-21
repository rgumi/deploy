<template>
  <div class="wrapper elevation-3">
    <!-- Info row -->
    <v-row fluid no-gutters dense>
      <div style="min-width: 15%; text-align:left">
        <h1 @dblclick="show = !show" >
          {{ route.name }}
        </h1>
      </div>
      <v-icon
        title="Go to switchover"
        @click="$router.push(switchoverLink)"
        :color="switchoverStatusColor"
        style="margin-left: 2px"
        v-if="route.switchover !== null"
        >mdi-swap-horizontal</v-icon
      >
      <span
        title="Currently active alerts"
        @click="editRoute()"
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
        title="Toggle visibility of routes"
        @click="show = !show"
        class="buttonIcon"
        v-bind:class="{ rotate: !show }"
        >mdi-arrow-down-bold-circle</v-icon
      >
    </v-row>
    <v-divider></v-divider>

    <v-row fluid no-gutters dense v-if="show">
      <v-col>
        <!-- Properties -->
        <v-row v-if="showConfig" fluid no-gutters class="text text-left">
          <v-col>
            <table class="props" style="width:100%">
              <tr>
                <th>Prefix</th>
                <td>{{ route.prefix }}</td>
              </tr>
              <tr>
                <th>Rewrite</th>
                <td>{{ route.rewrite }}</td>
              </tr>
              <tr>
                <th>Host</th>
                <td>{{ route.host }}</td>
              </tr>
              <tr>
                <th>Strategy</th>
                <td>{{ route.strategy.type }}</td>
              </tr>
              <tr>
                <th>Healthcheck</th>
                <td>{{ route.healthcheck_bool ? "active" : "inactive" }}</td>
              </tr>
              <tr>
                <th>Proxy</th>
                <td :contenteditable="editable" @blur="onEdit" title="proxy">
                  {{ route.proxy == "" ? "not defined" : route.proxy }}
                </td>
              </tr>
              <tr>
                <th>Methods</th>
                <td>
                  {{ route.methods }}
                </td>
              </tr>
            </table>
          </v-col>
          <v-col>
            <table class="props" style="width:100%">
              <tr>
                <th>CookieTTL (in min)</th>
                <td
                  :contenteditable="editable"
                  @blur="onEdit"
                  title="cookie_ttl"
                >
                  {{ route.cookie_ttl }}
                </td>
              </tr>
              <tr>
                <th>Read Timeout</th>
                <td :contenteditable="editable" @blur="onEdit" title="read_timeout">
                  {{ route.read_timeout }}
                </td>
              </tr>
              <tr>
                <th>Write Timeout</th>
                <td :contenteditable="editable" @blur="onEdit" title="write_timeout">
                  {{ route.write_timeout }}
                </td>
              </tr>
              <tr>
                <th>Idle Timeout</th>
                <td
                  :contenteditable="editable"
                  @blur="onEdit"
                  title="idle_timeout"
                >
                  {{ route.idle_timeout }}
                </td>
              </tr>
              <tr>
                <th>Scrape Interval</th>
                <td
                  :contenteditable="editable"
                  @blur="onEdit"
                  title="scrape_interval"
                >
                  {{ route.scrape_interval }}
                </td>
              </tr>
              <tr>
                <th>Monitoring Interval</th>
                <td :contenteditable="editable" @blur="onEdit" title="monitoring_interval">
                  {{ route.monitoring_interval }}
                </td>
              </tr>
              <tr>
                <th>Healthcheck Interval</th>
                <td :contenteditable="editable" @blur="onEdit" title="healthcheck_interval">
                  {{ route.healthcheck_interval }}
                </td>
              </tr>
            </table>
          </v-col>
        </v-row>

        <v-divider v-if="showBackends"></v-divider>
        <!-- Backends -->
        <div v-if="showBackends">
          <v-row fluid no-gutters class="text" style="padding-bottom: 0;">
            <h1>Backends</h1>
          </v-row>
          <v-row fluid class="text">
            <v-data-table
              :headers="headers"
              :items="backendsAsList"
              :search="search"
              :hide-default-footer="true"
              :disable-pagination="true"
              style="width: 100%"
            >
              <template v-slot:item="row">
                <tr @dblclick="gotoBackend">
                  <td class="left">
                    <v-icon v-if="row.item.active" color="green"
                      >mdi-check-circle</v-icon
                    >
                    <v-icon v-if="!row.item.active" color="red"
                      >mdi-close-circle</v-icon
                    >
                  </td>
                  <td class="left">{{ row.item.name }}</td>
                  <td class="left">{{ row.item.weight }}</td>
                  <td class="left">{{ row.item.addr }}</td>
                  <td class="left">{{ row.item.id }}</td>
                </tr>
              </template>
            </v-data-table>
          </v-row>
        </div>

        
        <div v-if="showSwitchover">
          <v-divider v-if="showSwitchover && route.switchover != null"></v-divider>
          <v-row fluid no-gutters class="text" style="padding-bottom: 0;">
              <Switchover v-if="route.switchover != null" :switchover="route.switchover" />
          </v-row>
        </div>

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
    <v-divider v-if="showButtons"></v-divider>
    <!-- Buttons -->
    <v-row fluid no-gutters dense v-if="showButtons">
      <v-btn class="ref-left" :to="routeLink">Configuration</v-btn>
      <v-btn class="ref-right" :to="dashboadLink">Dashboard</v-btn>
    </v-row>
  </div>
</template>

<script>
import AlertList from "@/components/AlertList";
import Switchover from "@/components/Switchover";
import NotificationBell from "vue-notification-bell";
export default {
  name: "routeComponent",
  components: {
    NotificationBell,
    AlertList,
    Switchover
  },
  props: {
    route: Object,
    showAll: Boolean,
    showButtons: Boolean,
    showSwitchover: Boolean,
    showBackends: Boolean,
    showAlerts: Boolean,
    showConfig: Boolean,
    editable: Boolean
  },
  created() {
    this.show = this.showAll;
  },
  data() {
    return {
      show: Boolean,
      search: "",
      headers: [
        {
          text: "Status",
          sortable: true,
          value: "status"
        },
        {
          text: "Name",
          sortable: true,
          value: "name"
        },
        {
          text: "Weight",
          sortable: true,
          value: "weight"
        },
        {
          text: "Address",
          sortable: false,
          value: "addr"
        },
        {
          text: "ID",
          sortable: false,
          value: "id"
        }
      ],
      backends: this.backendsAsList
    };
  },
  watch: {
    showAll() {
      this.show = this.showAll;
    }
  },
  computed: {
    activeAlerts: function() {
      return this.$store.getters.getActiveAlerts(this.route.name);
    },
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
      return this.$store.getters.getLoading;
    },
    switchoverStatusColor: function() {
      if (this.route.switchover !== null) {
        if (this.route.switchover.status == "Failed") {
          return "error"
        } else if (this.route.switchover.status == "Running") {
          return "success"
        } else if (this.route.switchover.status == "Stopped") {
          return "success"
        }
        return "success";
      }
      return "success";
    },
    backendsAsList: function() {
      if (this.route.backends === undefined) {
        return [];
      }
      var list = Array.from(Object.entries(this.route.backends).map(d => d[1]));
      list.forEach(b => {
        if (b.active) {
          b.status = "{{<v-icon>mdi-bookmark-check</v-icon> }}";
        } else {
          b.status = "{{<v-icon>mdi-bookmark-remove</v-icon> }}";
        }
      });
      return list;
    },
    activeAlertsCount: function() {
      return this.$store.getters.getActiveAlerts(this.route.name).length;
    }
  },
  methods: {
    gotoBackend: function() {
      alert(`Goto Backend`);
    },
    onEdit: function(e) {
      console.log(e);
      console.log(e.target.title);
      console.log(this.route.cookie_ttl);
    }
  }
};
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
