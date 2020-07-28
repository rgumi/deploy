<template>
  <v-app>
    <NavBar />
    <v-main>
      <v-btn
        @click="$router.go(-1)"
        style="z-index: 10;"
        absolute
        depressed
        :elevation="0"
        link
        outlined
        tile
      >
        <v-icon color="rgba(0, 0, 0, 1)" :size="25">mdi-arrow-left-bold</v-icon>
      </v-btn>
      <router-view keep-alive style="width: 80%;"></router-view>
    </v-main>
  </v-app>
</template>

<script>
//import NavDrawer from "./components/NavDrawer";
import NavBar from "./components/NavBar";
export default {
  name: "App",
  components: {
    NavBar
  },
  data: () => ({
    //
  })
};
</script>
