import Vue from "vue";
import VueRouter from "vue-router";
//import Home from "@/views/Home.vue";
import Dashboard from "@/views/Dashboard.vue";
import Routes from "@/views/Routes.vue";
import Route from "@/views/Route.vue";

Vue.use(VueRouter);

const routes = [
  {
    path: "/",
    name: "Home",
    component: Dashboard,
  },
  {
    path: "/dashboard",
    name: "Dashboard",
    component: Dashboard,
  },
  {
    path: "/routes",
    name: "Routes",
    component: Routes,
  },
  {
    path: "/routes/:name",
    name: "SpecificRoute",
    component: Route,
  },
  {
    path: "/about",
    name: "About",
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import("../views/About.vue"),
  },
  {
    path: "/help",
    name: "Help",
    component: () => import("../views/Help.vue"),
  },
  {
    path: "/login",
    name: "Login",
    component: () => import("../views/Login.vue"),
  },

  {
    path: "*",
    name: "NotFound",
    component: () => import("../views/PageNotFound.vue"),
  },
];

const router = new VueRouter({
  mode: "history",
  base: process.env.BASE_URL,
  routes,
});

export default router;
