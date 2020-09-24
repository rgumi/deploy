<template>
  <v-app>
    <NavBar />
    <v-main>
      <ButtonText
        style="top: 10px; margin-left: 10px"
        onHoverText="Back"
        btnIcon="mdi-arrow-left-bold"
        btnEvent="backEvent"
        @backEvent="$router.go(-1)"
      ></ButtonText>
      <transition>
        <keep-alive :max="5">
          <router-view keep-alive style="width: 80%;"></router-view>
        </keep-alive>
      </transition>
    </v-main>

    <EventBar></EventBar>
  </v-app>
</template>

<script>
//import NavDrawer from "./components/NavDrawer";
import NavBar from "./components/NavBar";
import EventBar from "./components/EventHandling/EventBar";
import { eventBus } from "@/main";
import ButtonText from "@/components/ButtonText";
export default {
  name: "App",
  components: {
    NavBar,
    EventBar,
    ButtonText,
  },
  data: () => ({
    //
  }),
  beforeMount() {
    window.addEventListener("beforeunload", this.preventNav);
  },

  beforeDestroy() {
    window.removeEventListener("beforeunload", this.preventNav);
  },
  created() {
    this.$store.dispatch("startPulling");
    console.log("started");
  },
  methods: {
    emitEvent() {
      eventBus.$emit("showEvent", {
        icon: "mdi-alert",
        icon_color: "error",
        title: "Error 404",
        message: "Could not find the requested resource",
      });
    },
    preventNav(event) {
      event.preventDefault();
      event.returnValue = "";
    },
  },
};
</script>

<style>
.avoid-clicks {
  -moz-user-select: none;
  -khtml-user-select: none;
  -webkit-user-select: none;
  -ms-user-select: none;
  user-select: none;
}
#app {
  background: rgba(0, 0, 0, 0.2);
}
</style>
