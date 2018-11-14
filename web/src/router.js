import Vue from 'vue'
import Router from 'vue-router'
import Admin from './components/Admin.vue'
import AdvancedSearch from './components/AdvancedSearch.vue'

Vue.use(Router)

export default new Router({
  routes: [
    {
      path: '/',
      name: 'new',
      component: AdvancedSearch
    },
    {
      path: '/admin',
      name: 'admin',
      component: Admin
    },
  ]
})
