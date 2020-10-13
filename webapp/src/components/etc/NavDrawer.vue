<template>
  <div>
    <v-app-bar color="primary" dark>
      <v-app-bar-nav-icon v-on:click="drawer = true"></v-app-bar-nav-icon>
      <div class="d-flex align-center">
        <v-img
          alt="Vuetify Logo"
          class="shrink mr-2"
          contain
          src="https://cdn.vuetifyjs.com/images/logos/vuetify-logo-dark.png"
          transition="scale-transition"
          width="40"
        />

        <v-img
          alt="Vuetify Name"
          class="shrink mt-1 hidden-sm-and-down"
          contain
          min-width="100"
          src="https://cdn.vuetifyjs.com/images/logos/vuetify-name-dark.png"
          width="100"
        />
      </div>

      <v-spacer></v-spacer>

      <v-menu left bottom>
        <template v-slot:activator="{ on, attrs }">
          <v-btn icon v-bind="attrs" v-on="on">
            <v-icon>mdi-dots-vertical</v-icon>
          </v-btn>
        </template>

        <v-list>
          <v-list-item v-for="n in 5" :key="n" @click="() => {}">
            <v-list-item-title>Option {{ n }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-app-bar>

    <v-navigation-drawer v-model="drawer" absolute temporary>
      <template v-slot:prepend>
        <v-list-item two-line v-on:click="getUser()">
          <v-list-item-avatar>
            <img :src="user.avatar" />
          </v-list-item-avatar>

          <v-list-item-content>
            <v-list-item-title>{{ user.name }}</v-list-item-title>
            <v-list-item-subtitle>{{ user.status }}</v-list-item-subtitle>
          </v-list-item-content>
        </v-list-item>
      </template>
      <v-divider></v-divider>

      <v-list nav dense>
        <v-list-item v-for="item in items" :key="item.title" link>
          <router-link :to="item.uri">
            <v-list-item-icon>
              <v-icon>{{ item.icon }}</v-icon>
            </v-list-item-icon>
            <NavLink>{{ item.title }}</NavLink>
          </router-link>
        </v-list-item>
      </v-list>
    </v-navigation-drawer>
  </div>
</template>

<script>
export default {
  name: "NavDrawer",
  data: () => ({
    drawer: false,
    user: {
      avatar: require("../assets/account_circle-grey-24dp.svg"),
      name: "Test",
      status: "Logged in"
    },
    items: [
      { title: "Home", uri: "/", icon: "mdi-home" },
      { title: "Dashboard", uri: "/dashboard", icon: "mdi-view-dashboard" },
      { title: "Routes", uri: "/routes", icon: "mdi-image" },
      { title: "Help", uri: "/help", icon: "mdi-help-box" },
      { title: "About", uri: "/about", icon: "mdi-help-box" }
    ]
  }),
  methods: {
    getUser: () => {
      console.log("Redirecting to user page");
    }
  }
};
</script>
