<template>
  <div class="right bar">
    <v-list style="background-color: transparent;">
      <v-list-item v-for="(event, index) in events" :key="index">
        <EventTile
          style="padding: 0; margin-bottom: 5px;"
          :input="event"
          :index="index"
          @removeEvent="removeMe(index)"
        ></EventTile>
      </v-list-item>
    </v-list>
  </div>
</template>

<script>
import EventTile from "./EventTile";
import { eventBus } from "@/main";

export default {
  name: "EventBar",
  components: {
    EventTile
  },
  data: () => ({
    events: []
  }),
  created() {
    eventBus.$on("showEvent", event => {
      console.log(event);
      this.events.push(event);
    });
  },
  methods: {
    removeMe(index) {
      this.events.splice(index, 1);
    }
  },
  watch: {
    events() {
      if (this.events.length > 7) {
        this.events.shift();
      }
    }
  }
};
</script>

<style scoped>
.right {
  position: fixed;
  float: right;
  z-index: 10;
  right: 0;
  top: 0;
}

.bar {
  background-color: transparent;
  display: fixed;
  margin-top: 8vh;
  opacity: 0.8;
  padding: 0;
}
</style>
