import Vue from 'vue'
import Antd from 'ant-design-vue';
import App from './App.vue'
import 'ant-design-vue/dist/antd.css';
import VueRouter from 'vue-router'
import routers from './routers.js'
import axios from 'axios'
import VueAxios from 'vue-axios'


Vue.config.productionTip = false
Vue.use(VueAxios, axios);
Vue.use(VueRouter)
Vue.use(Antd)
Vue.prototype.$baseUrl = 'http://192.168.56.144:8090'
Vue.prototype.$baseWsUrl = 'ws://192.168.56.144:8090'
const router = new VueRouter({
  mode: 'history',
  routes: routers
})

new Vue({
  el: '#app',
  router,
  render: h => h(App),
}).$mount('#app')
