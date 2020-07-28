import Vue from "vue";
import Vuex from "vuex";

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    iconList: {
      running: {
        color: "success",
        icon: "mdi-run",
      },
      idle: {
        color: "grey",
        icon: "mdi-run",
      },
    },
  },
  mutations: {},
  actions: {},
  modules: {},
  getters: {
    getIcon: (state) => (status) => state.iconList[status],
  },
});
