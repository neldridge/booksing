import Vue from 'vue'
import Router from 'vue-router'
import Home from './components/Home.vue'
import AdvancedSearch from './components/AdvancedSearch.vue'

Vue.use(Router)

export default new Router({
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home
    },
    {
      path: '/new',
      name: 'about',
      component: AdvancedSearch
    },
  ]
})
