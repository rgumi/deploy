<template>
  <!-- 
    => Consists of a title, an icon/symbol, button to remove and a text 
    => After 2 secs delay fade away, on hover pause the timer.
  -->
  <div>
    <v-container fluid class="tile avoid-clicks hidden-xs-only" @click="remove">
      <v-row justify="center" align="center" dense>
        <v-col class="icon_col">
          <v-icon size="42" :color="input.icon_color">{{ input.icon }}</v-icon>
        </v-col>
        <v-col>
          <h1>{{ input.title }}</h1>
          <p>{{ input.message }}</p>
        </v-col>
      </v-row>
    </v-container>

    <div class="hidden-sm-and-up" @click="remove">
      <v-row @mouseleave="show = false">
        <v-icon size="42" :color="input.icon_color" @mouseover="show = true">
          {{
          input.icon
          }}
        </v-icon>
        <v-col v-if="show">
          <h1>{{ input.title }}</h1>
          <p>{{ input.message }}</p>
        </v-col>
      </v-row>
    </div>
  </div>
</template>

<script>
export default {
  name: "eventTile",
  props: {
    input: Object,
    removeInterval: {
      type: Number,
      default: 5000
    }
  },
  data: () => {
    return {
      show: false
    };
  },
  methods: {
    remove() {
      this.$emit("removeEvent");
    }
  },
  created() {
    setTimeout(() => this.remove(), this.removeInterval);
  }
};
</script>

<style scoped>
.tile {
  width: 250px;
  min-height: 100px;
  background-color: rgba(0, 0, 0, 0.2);
}
.tile:hover {
  background-color: rgba(255, 1, 1, 0.568);
}
h1 {
  font-size: 2vh;
}
.text {
  font-weight: 500;
  font-size: 2vh;
}
.icon_col {
  max-width: 60px;
}
</style>
