<template>
  <div id="log-block">
    <p>
      <a-tag
        color="#69b3b59c"
        style="
          font-size: 14px;
          padding: 2px;
          padding-left: 5px;
          padding-right: 5px;
        "
        >{{ this.time }}</a-tag
      >

      <a-tooltip>
        <template slot="title">
          <span>回滚到该日志之后</span>
        </template>
        <a-button shape="circle" type="danger" @click="handleRollBack">
          <a-icon type="undo" />
        </a-button>
      </a-tooltip>
    </p>

    <p>
      用户：
      <a-tag
        color="#a6d7da"
        style="
          font-size: 14px;
          padding: 2px;
          padding-left: 5px;
          padding-right: 5px;
        "
        >{{ name }}</a-tag
      >对单元格
      <a-tag
        color="#a6d7da"
        style="
          font-size: 14px;
          padding: 2px;
          padding-left: 5px;
          padding-right: 5px;
        "
        >{{ r }} 行 {{ c }} 列</a-tag
      >进行了修改
    </p>

    <div
      v-if="this.v"
      style="
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 430px;
      "
    >
      现值为：
      {{ this.json.v.v }}
      <br />
      <br />
    </div>

    <div
      v-if="this.bg | this.fc | this.fs | this.ff | this.cl | this.it | this.bl"
    >
      <span>属性有：</span>
      <span v-if="this.bg">
        背景颜色
        <a-tooltip>
          <template slot="title">{{ this.json.v.bg }}</template>
          <a-icon
            type="tablet"
            theme="twoTone"
            :two-tone-color="this.json.v.bg"
          />
        </a-tooltip>
      </span>
      <span v-if="this.fc">
        文字颜色
        <a-tooltip>
          <template slot="title">{{ this.json.v.fc }}</template>
          <a-icon
            type="tablet"
            theme="twoTone"
            :two-tone-color="this.json.v.fc"
          />
        </a-tooltip>
      </span>
      <span v-if="this.fs">字号为{{ this.json.v.fs }}</span>
      <span v-if="this.ff">字体为{{ this.json.v.ff }}</span>
      <span v-if="this.cl">删除线</span>
      <span v-if="this.it">斜体</span>
      <span v-if="this.bl">粗体</span>
      <br />
      <br />
    </div>

    <vue-json-pretty :data="json" :showLine="false" :deep="2"></vue-json-pretty>
  </div>
</template>

<script>
import Icon from "@ant-design/icons";
import { defineComponent } from "vue";
import { Options, Vue } from "vue-class-component";
import VueJsonPretty from "vue-json-pretty";
import "vue-json-pretty/lib/styles.css";
import axios from "axios";

export default {
  name: "LogBlock",
  components: {
    VueJsonPretty,
  },
  props: {
    json: Object,
    name: String,
    date: String,
    // filename: String
  },
  data() {
    return {
      time: null,
      c: null,
      r: null,
      v: null,
      bg: null,
      ff: null,
      fc: null,
      bl: null,
      it: null,
      fs: null,
      cl: null,
      rollbackPath: this.$baseUrl + "/rollback",
    };
  },
  mounted() {
    let time = new Date(this.date);
    let Y = time.getFullYear() + " - ";
    let M =
      (time.getMonth() + 1 < 10
        ? "0" + (time.getMonth() + 1)
        : time.getMonth() + 1) + " - ";
    let D = time.getDate() + " ";
    let h = time.getHours() + ":";
    let m = time.getMinutes() + ":";
    let s = time.getSeconds();
    this.time = Y + M + D + h + m + s;

    this.c = this.json.c;
    this.r = this.json.r;

    if (this.json.v.hasOwnProperty("v")) {
      this.v = true;
    }
    if (this.json.v.hasOwnProperty("bg")) {
      this.bg = true;
    }
    if (this.json.v.hasOwnProperty("ff")) {
      this.ff = true;
    }
    if (this.json.v.hasOwnProperty("fc")) {
      this.fc = true;
    }
    if (this.json.v.hasOwnProperty("bl")) {
      this.bl = true;
    }
    if (this.json.v.hasOwnProperty("it")) {
      this.it = true;
    }
    if (this.json.v.hasOwnProperty("fs")) {
      this.fs = true;
    }
    if (this.json.v.hasOwnProperty("cl")) {
      this.cl = true;
    }
  },
  methods: {
    handleRollBack() {
      axios
        .post(this.rollbackPath, {
          filename: this.$route.params.filename,
          timestamp: this.date,
        })
        .then((response) => {
          if (response.data) {
            this.$message.success("回滚成功");
            this.$router.push({
              path: `/file/${this.$route.params.filename}`,
            });
          } else {
            this.$message.error("回滚失败");
          }
        });

      // this.normalFileList.splice(index, 1);
      // this.deletedFileList.push(item);
    },
  },
};
</script>

<style >
#log-block {
  min-width: 510px;
  text-align: left;
  padding-left: 30px;
  /* background-color: #69b3b59c; */
  padding-top: 10px;
  padding-bottom: 10px;
  padding-right: 10px;
  border-radius: 10px;
  margin: 0 auto;
  margin-left: 20px;
  box-shadow: 2px 4px 5px 2px #dcdfdf;
}

.ant-btn-danger {
  color: #fff;
  background-color: #e69560;
  border-color: #e69560;
  text-shadow: 0 -1px 0 rgb(0 0 0 / 12%);
  box-shadow: 0 2px 0 rgb(0 0 0 / 5%);
}
</style>