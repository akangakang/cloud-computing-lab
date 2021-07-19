<template>
  <a-locale-provider :locale="locale">
    <div id="app">
      <router-view></router-view>
    </div>
  </a-locale-provider>
</template>

<script>
import Hello from "./views/Hello.vue";
import zhCN from "ant-design-vue/lib/locale-provider/zh_CN";
export default {
  name: "App",
  components: {
    Hello
  },
  data() {
    return {
      locale: zhCN
    };
  }
};
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  height: 100%;
  background-color: #edeeed;
  /* margin-top: 60px; */
}
</style>
