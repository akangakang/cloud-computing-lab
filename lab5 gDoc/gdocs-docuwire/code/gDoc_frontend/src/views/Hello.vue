<template>
  <div id="login-block">
    <a-row justify="center">
      <a-col>
        <span id="login-font">DocuWire</span>
        <br />
        <br />
        <a-form :model="formInline" @submit="handleSubmit" @submit.native.prevent>
          <a-form-item class="login-input">
            <a-input v-model:value="formInline.name" placeholder="请输入用户名">
              <template #prefix>
                <div style="color:rgba(0,0,0,.25)" />
              </template>
            </a-input>
          </a-form-item>

          <br />

          <a-form-item>
            <a-button html-type="submit" :disabled="formInline.name === ''" @click="goToFile">进入文件列表</a-button>
          </a-form-item>
        </a-form>
      </a-col>
    </a-row>
  </div>
</template>

<script>
import Icon from "@ant-design/icons";
import { defineComponent } from "vue";
import { Options, Vue } from "vue-class-component";
import { Button } from "ant-design-vue";
import { message } from "ant-design-vue";
export default {
  name: "Hello",
  components: {
    "a-button": Button
  },
  data() {
    return {
      formInline: {
        name: ""
      }
    };
  },
  methods: {
    handleSubmit(e) {
      console.log(this.formInline);
    },
    goToFile() {
      localStorage.setItem("uname", this.formInline.name);
      this.$router.push({
        path: `/filelist`
      });
    }
  }
};
</script>

<style >
#login-block {
  top: 26%;
  position: fixed;
  left: 33%;
  text-align: center;
  background-color: #ffffff;
  width: 480px;
  border-radius: 40px;
  padding: 20px;
  margin: 0 auto;
  box-shadow: 3px 3px 2px #dcdfdf;
}
#login-font {
  font-size: 35px;
  color: #485355f5;
  font-weight: bold;
}
.login-input .ant-input-affix-wrapper > input.ant-input {
  padding: 0;
  border: none;
  outline: none;
  background-color: #bae7e500;
}
.login-input .ant-input-affix-wrapper {
  border: 1px solid #ffffff00;
  border-radius: 12px;
  padding: 4px 14px;
  width: 80%;
  background-color: #bae7e5d3;
}
</style>