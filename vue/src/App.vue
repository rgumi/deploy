<template>
  <v-app>
    <NavBar />
    <v-main>
      <router-view keep-alive></router-view>
    </v-main>
  </v-app>
</template>

<script>
//import NavDrawer from "./components/NavDrawer";
import NavBar from "./components/NavBar";
export default {
  name: "App",
  components: {
    NavBar,
  },
  data: () => ({
    //
  }),
};
</script>
