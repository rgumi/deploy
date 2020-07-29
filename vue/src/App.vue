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

    <EventBar></EventBar>
  </v-app>
</template>

<script>
//import NavDrawer from "./components/NavDrawer";
import NavBar from "./components/NavBar";
import EventBar from "./components/EventHandling/EventBar";
import { eventBus } from "@/main";
export default {
  name: "App",
  components: {
    NavBar,
    EventBar
  },
  data: () => ({
    //
  }),
  methods: {
    emitEvent() {
      eventBus.$emit("showEvent", {
        icon: "mdi-alert",
        icon_color: "error",
        title: "Error 404",
        message: "Could not find the requested resource"
      });
    }
  }
};
</script>
