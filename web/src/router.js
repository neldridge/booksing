import Vue from "vue";
import Router from "vue-router";
import Admin from "./components/Admin.vue";
import AdvancedSearch from "./components/AdvancedSearch.vue";

Vue.use(Router);

export default new Router({
  mode: "history",
  routes: [
    {
      path: "/",
      name: "home",
      component: AdvancedSearch
    },
    {
      path: "/admin",
      name: "admin",
      component: Admin
    }
  ]
});
