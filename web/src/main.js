import "./../node_modules/bulma/css/bulma.css";
import "vue-material-design-icons/styles.css";
import Vue from "vue";
import App from "./App.vue";
import Buefy from "buefy";
import router from "./router";
import store from "./store";
import config from "./firebase-config";
import visibility from "vue-visibility-change";

// Firebase App (the core Firebase SDK) is always required and must be listed first
import * as firebase from "firebase/app";
import "firebase/auth";
import "firebase/firestore";

Vue.use(Buefy);
Vue.use(visibility);

Vue.config.productionTip = false;

// Initialize Firebase
firebase.initializeApp(config);

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount("#app");

// Install servicerWorker if supported.
if ("serviceWorker" in navigator) {
  navigator.serviceWorker
    .register("/serviceworker.js", { scope: "/" })
    .then(reg => {
      // Registration worked.
      console.log("Registration succeeded. Scope is " + reg.scope);
    })
    .catch(error => {
      // Registration failed.
      console.log("Registration failed with " + error.message);
    });
}
