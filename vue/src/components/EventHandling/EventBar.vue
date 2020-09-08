<template>
  <div class="right bar   ">
    <v-list style="padding: 0;">
      <v-list-item v-for="(event, index) in events" :key="index">
        <EventTile
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
    deleteEvents() {
      if (this.events[8]) {
        this.events.shift();
      }
    },
    removeMe(index) {
      this.events.splice(index, 1);
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
  display: fixed;
  margin-top: 8vh;
  height: 100%;
  opacity: 0.8;
  padding: 0;
}
</style>
