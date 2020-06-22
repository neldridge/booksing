import Vue from "vue";
import Vuex from "vuex";
import VuexPersistence from "vuex-persist";

const vuexLocal = new VuexPersistence({
  key: "example",
  storage: window.localStorage
});

Vue.use(Vuex);

const store = new Vuex.Store({
  state: {
    username: "",
    token: "",
    authenticated: false
  },
  getters: {
    username: state => state.username,
    authenticated: state => state.authenticated
  },
  mutations: {
    setUsername: (state, username) => {
      state.username = username;
    },
    setAuthenticated: (state, valid) => {
      state.authenticated = valid;
    }
  },
  actions: {
    login: (context, payload) => {
      context.commit("setUsername", payload.username);
      context.commit("setAuthenticated", true);
    },
    logout: context => {
      context.commit("setAuthenticated", false);
    }
  },
  plugins: [vuexLocal.plugin]
});

export default store;
