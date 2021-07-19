<template>
  <div>
    <a-row style="margin-top:3%">
      <a-col :span="4" :offset="3">
        <div class="log-subtitle">{{ this.$route.params.filename}}</div>
        <div class="log-title">日 志</div>
      </a-col>
      <a-col :span="8" :offset="1">
        <a-timeline v-if="this.logdata != null">
          <a-timeline-item color="gray" v-for="item in logdata" :key="item.timestamp">
            <log-block :date="item.timestamp" :name="item.user" :json="item.value" />
          </a-timeline-item>
        </a-timeline>
        <a-empty v-else />
      </a-col>
    </a-row>
  </div>
</template>

<script>
import { defineComponent } from "vue";
import { Options, Vue } from "vue-class-component";
import axios from "axios";
import LogBlock from "../components/LogBlock.vue";

export default {
  components: {
    // VueJsonPretty,
    LogBlock
  },
  data() {
    return {
      json: { d: "sss" },
      logdata: [],
      logPath: this.$baseUrl + "/getLog",
      
    };
  },
  mounted() {
    axios
      .get(this.logPath, {
        params: {
          filename: this.$route.params.filename
        }
      })
      .then(response => {
        this.logdata = response.data;

        if(this.logdata!=null)
        {
          this.logdata.reverse();
        }
        
      });
  }
};
</script>

<style>
.log-title {
  font-size: xx-large;
  font-weight: bold;
  line-height: 1.5em;
  float: right;
  font-size: 30px;
  /* width: 1.5em; */
  word-wrap: break-word;
  writing-mode: vertical-lr;
  word-break: break-all;
  background-color: #4e97999c;
  width: auto;
  border-radius: 10px;
  margin: 0 auto;
  box-shadow: 3px 3px 2px #dcdfdf;
  color: #f2f4f4;
  padding: 8px;
  padding-top: 15px;
  padding-bottom: 15px;
}

.log-subtitle {
  font-size: small;
  float: right;
  font-size: 15px;
  width: 1.5em;
  word-wrap: break-word;
  writing-mode: vertical-lr;
  word-break: break-all;
  background-color: #4e97999c;
  width: auto;
  border-radius: 10px;
  margin-left: 10px;
  box-shadow: 3px 3px 2px #dcdfdf;
  color: #f2f4f4;
  padding-left: 4px;
  padding-right: 4px;
  padding-bottom: 12px;
  padding-top: 12px;
}
</style>