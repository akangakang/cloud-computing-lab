<template>
  <div>
    <a-page-header
      style="padding: 4px; padding-left: 20px;"
      title="DocuWire"
      :sub-title="this.filename"
    >
      <template #extra>
        <a-button key="1" type="primary">
          <a-icon type="user" />
          {{uname}}
        </a-button>
      </template>
    </a-page-header>
    <div
      id="luckysheet"
      style="margin:0px;padding:0px;position:absolute;width:100%;left: 0px;top:38px;bottom:0px;"
    ></div>
  </div>
</template>

<script>
import LuckyExcel from "luckyexcel";

export default {
  name: "LuckyExcel",
  data() {
    return {
      filename: null,
      uname: null,
      // httppath: "http://192.168.56.137/",
      loadpath: this.$baseUrl + "/load?filename=",
      connectpath: this.$baseWsUrl + "/connect?name="
    };
  },
  mounted() {
    this.init();
  },
  methods: {
    init() {
      this.uname = localStorage.getItem("uname");
      this.filename = this.$route.params.filename;
      console.log("uname", this.uname, "filename", this.filename);
      let lp = this.loadpath + this.filename;
      let cp = this.connectpath + this.uname + "&filename=" + this.filename;
      let uname = this.uname;
      let filename = this.filename;

      var options = {
        container: "luckysheet", // 设定DOM容器的id
        title: filename, // 设定表格名称
        lang: "zh", // 设定表格语言
        allowUpdate: true,
        loadUrl: lp,
        updateUrl: cp,
        userInfo: uname,
        cellRightClickConfig: {
          copy: true, // 复制
          rowHeight: true, // 行高
          columnWidth: true, // 列宽
          copyAs: false, // 复制为
          paste: false, // 粘贴
          insertRow: false, // 插入行
          insertColumn: false, // 插入列
          deleteRow: false, // 删除选中行
          deleteColumn: false, // 删除选中列
          deleteCell: false, // 删除单元格
          hideRow: false, // 隐藏选中行和显示选中行
          hideColumn: false, // 隐藏选中列和显示选中列
          clear: false, // 清除内容
          matrix: false, // 矩阵操作选区
          sort: false, // 排序选区
          filter: false, // 筛选选区
          chart: false, // 图表生成
          image: false, // 插入图片
          link: false, // 插入链接
          data: false, // 数据验证
          cellFormat: false // 设置单元格格式
        },
        showsheetbarConfig: {
          add: false, //新增sheet
          menu: false, //sheet管理菜单
          sheet: false //sheet页显示
        },
        showtoolbarConfig: {
          undoRedo: false, //撤销重做，注意撤消重做是两个按钮，由这一个配置决定显示还是隐藏
          paintFormat: false, //格式刷
          currencyFormat: false, //货币格式
          percentageFormat: false, //百分比格式
          moreFormats: false, // '更多格式'
          textWrapMode: false, // '换行方式'
          textRotateMode: false, // '文本旋转方式'
          image: false, // '插入图片'
          link: false, // '插入链接'
          chart: false, // '图表'（图标隐藏，但是如果配置了chart插件，右击仍然可以新建图表）
          postil: false, //'批注'
          pivotTable: false, //'数据透视表'
          function: false, // '公式'
          frozenMode: false, // '冻结方式'
          sortAndFilter: false, // '排序和筛选'
          conditionalFormat: false, // '条件格式'
          dataVerification: false, // '数据验证'
          splitColumn: false, // '分列'
          screenshot: false, // '截图'
          findAndReplace: false, // '查找替换'
          protection: false, // '工作表保护'
          print: false ,// '打印'
          font: false,
        },
        showinfobar: false,
        enableAddRow: false,
        sheetFormulaBar: false,
        count: false, // 计数栏
        view: false, // 打印视图
        zoom: false // 缩放
      };

      luckysheet.create(options);
    }
  },
  destroyed() {
    // 销毁监听
    this.socket.onclose = this.close;
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style >
h3 {
  margin: 40px 0 0;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}

.ant-page-header-heading-sub-title {
  float: left;
  /* margin-top: 10px; */
  margin: 7px 0;
  /* margin-right: 12px; */
  color: rgba(0, 0, 0, 0.45);
  font-size: 14px;
  line-height: 22px;
}

.ant-btn-primary {
  background-color: #69c8c3;
  border-color: #c6ebe9;
  text-shadow: 0 -1px 0 rgb(0 0 0 / 12%);
  box-shadow: 0 2px 0 rgb(0 0 0 / 5%);
}

.ant-btn-primary:hover,
.ant-btn-primary:focus {
  color: #fff;
  background-color: #2ed3c5;
  border-color: #2ed3c5;
}
.ant-page-header-heading {
  width: 99%;
  overflow: hidden;
}
</style>
