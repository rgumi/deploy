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
    ButtonText
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
        message: "Could not find the requested resource"
      });
    },
    preventNav(event) {
      event.preventDefault();
      event.returnValue = "";
    }
  }
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

.buttonIcon {
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
.v-data-table-header th {
  text-align: center;
}

.notification-text {
  background: rgba(223, 35, 35, 0.8);
  border-radius: 15%;
  height: 30px;
  width: 50px;
}
.left {
  text-align: left;
}
table tr:last-child th {
  border-bottom: none;
}

table tr:last-child td {
  border-bottom: none;
}

.props tr td {
  font-size: 0.875rem;
  height: 48px;
  border-bottom: thin solid rgba(0, 0, 0, 0.12);
  padding: 12px;
}
.props tr th {
  width: 150px;
  font-size: 0.875rem;
  height: 48px;
  border-bottom: thin solid rgba(0, 0, 0, 0.12);
  border-right: thin solid rgba(0, 0, 0, 0.12);
  padding: 12px;
}
.props tr {
  border-spacing: 0;
  border-collapse: separate;
  display: table-row;
  vertical-align: inherit;
  border-color: inherit;
}

.method-wrapper {
  background: rgba(0, 0, 0, 0.2);
  margin-bottom: 5px;
  padding: 5px;
  border-radius: 15%;
  margin-right: 5px;
}
</style>
