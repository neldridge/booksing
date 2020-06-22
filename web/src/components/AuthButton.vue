<template>
  <div>
    <button v-if="loading" class="button is-loading is-medium">Loading</button>
    <button v-else-if="!loggedin" class="button is-primary is-medium" @click="googleLogin">log in</button>
    <button v-else @click="logout" class="button is-danger is-medium">sign out</button>
  </div>
</template>

<script>
import axios from "axios";
import router from "../router";
import store from "../store";
// Firebase App (the core Firebase SDK) is always required and must be listed first
import * as firebase from "firebase/app";

import "firebase/auth";

export default {
  name: "AuthButton",
  components: {},

  data() {
    return {
      input: {
        username: "",
        password: ""
      },
      loggedin: false,
      isFullPage: false,
      loading: true,
      showWarning: false,
      warningMessage: "",
      lastUpdate: null
    };
  },
  mounted: function() {
    this.refreshAuth();
  },
  methods: {
    googleLogin: function() {
      var provider = new firebase.auth.GoogleAuthProvider();
      var vm = this;
      vm.loading = true;

      firebase
        .auth()
        .signInWithPopup(provider)
        .then(result => {
          // This gives you a Google Access Token. You can use it to access the Google API.
          var token = result.credential.accessToken;
          var idToken = result.credential.idToken;
          // The signed-in user info.
          var user = result.user;
          firebase
            .auth()
            .currentUser.getIdToken(/* forceRefresh */ true)
            .then(function(idToken) {
              store.dispatch("login", {
                username: user.email
              });
              vm.loading = false;
              vm.loggedin = true;
              vm.$emit("user-logged-in", true);
            });
        })
        .catch(function(error) {
          console.log("ERROR");
          // Handle Errors here.
          var errorCode = error.code;
          var errorMessage = error.message;
          // The email of the user's account used.
          var email = error.email;
          // The firebase.auth.AuthCredential type that was used.
          var credential = error.credential;
          // ...
          console.log(errorCode);
          console.log(errorMessage);
          console.log(email);
          console.log(credential);
          this.showErrorAlert(errorMessage);
          // ...
        });
    },
    logout() {
      var vm = this;
      this.$emit("user-logged-in", false);
      firebase
        .auth()
        .signOut()
        .then(function() {
          store.dispatch("logout", {});
          this.loading = false;
          this.loggedin = false;
        })
        .catch(function(error) {
          // An error happened
        });
    },
    refreshAuth: function() {
      var vm = this;
      vm.loading = true;
      firebase.auth().onAuthStateChanged(user => {
        if (user) {
          // User is signed in.
          this.loading = false;
          this.loggedin = true;
          vm.$emit("user-logged-in", true);
        } else {
          this.loading = false;
          this.loggedin = false;
        }
      });
    },

    showErrorAlert: function(msg) {
      this.$toast.open({
        duration: 5000,
        message: msg,
        type: "is-danger"
      });
    }
  }
};
</script>

<style>
.loginform {
  width: 640px;
}
</style>
