<template>
  <div>
    <v-navigation-drawer v-model="sidebar" absolute>
      <v-list nav tile>
        <v-list-item v-for="item in menuItems" :key="item.title" :to="item.path">
          <v-list-item-action>
            <v-icon>{{ item.icon }}</v-icon>
          </v-list-item-action>
          <v-list-item-content>{{ item.title }}</v-list-item-content>
        </v-list-item>
      </v-list>
    </v-navigation-drawer>
    <v-toolbar>
      <v-app-bar-nav-icon class="hidden-sm-and-up" @click="sidebar = !sidebar"></v-app-bar-nav-icon>
      <v-toolbar-title>
        <router-link to="/" tag="span" style="cursor: pointer">
          {{
          appTitle
          }}
        </router-link>
      </v-toolbar-title>
      <v-spacer></v-spacer>
      <v-toolbar-items class="hidden-xs-only">
        <v-btn text v-for="item in menuItems" :key="item.title" :to="item.path">
          <v-icon left>{{ item.icon }}</v-icon>
          {{ item.title }}
        </v-btn>

        <v-btn text @click="say('clicked')">
          <v-icon left dark>mdi-more</v-icon>Newsfeed
        </v-btn>
      </v-toolbar-items>
    </v-toolbar>
  </div>
</template>

<script>
export default {
  name: "App",
  data() {
    return {
      appTitle: "AutoDeploy",
      sidebar: false,
      menuItems: [
        { title: "Home", path: "/", icon: "mdi-home" },
        { title: "Dashboard", path: "/dashboard", icon: "mdi-view-dashboard" },
        { title: "Routes", path: "/routes", icon: "mdi-routes" }
      ]
    };
  },
  methods: {
    say: msg => {
      alert(msg);
    }
  }
};
</script>
