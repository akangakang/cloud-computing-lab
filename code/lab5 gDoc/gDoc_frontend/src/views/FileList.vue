<template>
  <div class="fileList">
    <a-row style="margin-top:3%">
      <a-col :span="18" :offset="3">
        <div
          class="fileList-title"
          style=" font-weight: 800; font-size: xx-large; color: #deeae8;"
        >DocuWire</div>
      </a-col>
    </a-row>
    <br />
    <a-row style="margin-top:3%">
      <a-col :span="18" :offset="3">
        <a-list
          :grid="{ gutter: 16, xs: 1, sm: 2, md: 4, lg: 4, xl: 6, xxl: 3 }"
          :data-source="normalFileList"
        >
          <a-list-item slot="renderItem" slot-scope="item, index">
            <a-card hoverable v-if="index==0" title="创建文件" :headStyle="tstyle">
              <template slot="actions" class="ant-card-actions">
                <div>
                  <a-tooltip>
                    <template slot="title">
                      <span>创建</span>
                    </template>
                    <a-button type="primary" shape="circle" @click="showModal">
                      <a-icon type="plus-circle" />
                    </a-button>
                  </a-tooltip>
                  <a-modal
                    title="请输入文件名"
                    :visible="visible"
                    :confirm-loading="confirmLoading"
                    @ok="handleCreate"
                    @cancel="handleCancel"
                  >
                    <a-input
                      ref="fileNameInput"
                      v-model="newfilename"
                      placeholder="只允许英文字母、数字以及 - 和 _ "
                      @pressEnter="handleCreate"
                    >
                      <a-icon type="snippets" slot="prefix" />
                      <a-tooltip slot="suffix" title="文件名只允许英文字母、数字以及 - 和 _ ">
                        <a-icon type="info-circle" style="color: rgba(0,0,0,.45)" />
                      </a-tooltip>
                    </a-input>
                  </a-modal>
                </div>
              </template>
            </a-card>

            <a-card hoverable v-else :title="item.title" :headStyle="tstyle">
              <template slot="actions" class="ant-card-actions">
                <a-tooltip>
                  <template slot="title">
                    <span>移入回收站</span>
                  </template>
                  <a-button shape="circle" type="danger" @click="handleDelete(item,index)">
                    <a-icon type="delete" />
                  </a-button>
                </a-tooltip>
                <a-tooltip>
                  <template slot="title">
                    <span>打开文件</span>
                  </template>
                  <a-button
                    type="primary"
                    shape="circle"
                    style="margin-left:10px"
                    @click="handleOpen(item)"
                  >
                    <a-icon type="folder-open" />
                  </a-button>
                </a-tooltip>
                <a-tooltip>
                  <template slot="title">
                    <span>查看日志</span>
                  </template>
                  <a-button
                    type="primary"
                    shape="circle"
                    style="margin-left:10px"
                    @click="handleLog(item)"
                  >
                    <a-icon type="pic-right" />
                  </a-button>
                </a-tooltip>
              </template>
            </a-card>
          </a-list-item>
        </a-list>
        <br />
        <div id="Recycled">
          <a-list
            :grid="{ gutter: 16, xs: 1, sm: 2, md: 4, lg: 4, xl: 6, xxl: 3 }"
            :data-source="deletedFileList"
          >
            <a-list-item slot="renderItem" slot-scope="item, index">
              <a-card hoverable :title="item.title" :headStyle="tstyle_delete">
                <template slot="actions" class="ant-card-actions">
                  <a-tooltip>
                    <template slot="title">
                      <span>恢复</span>
                    </template>
                    <a-button shape="circle" @click="handleRecover(item,index)">
                      <a-icon type="reload" />
                    </a-button>
                  </a-tooltip>
                  <a-tooltip>
                    <template slot="title">
                      <span>彻底删除</span>
                    </template>
                    <a-button shape="circle" @click="handleDeleteTrue(item,index)">
                      <a-icon type="rest" />
                    </a-button>
                  </a-tooltip>
                </template>
              </a-card>
            </a-list-item>
          </a-list>
        </div>
      </a-col>
    </a-row>
  </div>
</template>
<script>
import axios from "axios";

export default {
  name: "FileList",
  data() {
    return {
      tstyle: { color: "#252525", "font-weight": "600", "font-size": "large" },
      tstyle_delete: {
        color: "#898f8e",
        "font-weight": "600",
        "font-size": "large"
      },
      deletedFileList: [],
      normalFileList: [],
      uname: null,
      getFileListPath: this.$baseUrl + "/getFileList",
      createFilePath: this.$baseUrl + "/createFile",
      deleteFilePath: this.$baseUrl + "/deleteFile",
      deleteTrueFilePath: this.$baseUrl + "/deleteRecycled",
      recoverPath: this.$baseUrl + "/recoverFile",
      visible: false,
      confirmLoading: false,
      newfilename: null,
      fileList: []
    };
  },
  mounted() {
    this.uname = this.$route.params.uname;
    // console.log(this.getFileListPath);

    axios
      .get(this.getFileListPath)
      .then(response => {
        console.log("this.getFileListPath return")
        this.fileList = response.data;
        if (this.fileList != null) {
          for (const v of this.fileList) {
            if (v.isDeleted == false) {
              this.normalFileList.unshift(v);
            } else {
              this.deletedFileList.unshift(v);
            }
          }
        }
        this.normalFileList.unshift({ title: "创建文件", isDeleted: false });
      })
      .catch(function(error) {
        console.log(error);
      });

    
  },
  methods: {
    showModal() {
      this.visible = true;
    },
    handleCreate(e) {
      var reg = new RegExp(/^[-_0-9a-zA-Z]{1,}$/);
      if (reg.test(this.newfilename)) {
        let exist = false;
        // console.log(this.fileList)
        if (this.fileList != null) {
          this.fileList.find(item => {
            if (item.title == this.newfilename) {
              this.$message.error("文件已存在");
              exist = true;
            }
          });
        }

        if (!exist) {
          this.confirmLoading = true;
          axios
            .post(this.createFilePath, { filename: this.newfilename })
            .then(response => {
              let flag = response.data;
              console.log("createFile response",flag)
              if (flag) {
                this.visible = false;
                this.confirmLoading = false;
                this.normalFileList.splice(1, 0, {
                  title: this.newfilename,
                  isDeleted: false
                });
                this.newfilename = null;
              } else {
                this.$message.error("创建文件失败");
                this.confirmLoading = false;
              }
            })
            .catch(function(error) {
              console.log(error);
            });

          /* 以上注释和以下代码二选一 */
          // this.visible = false;
          // this.confirmLoading = false;
          // this.normalFileList.splice(1, 0, {
          //   title: this.newfilename,
          //   isDeleted: false
          // });
          // this.newfilename = null;
        }
      } else {
        this.$message.error("文件名只允许英文字母、数字以及 - 和 _ ");
      }
    },
    handleDelete(item, index) {
      axios
        .post(this.deleteFilePath, { filename: item.title })
        .then(response => {
          if (response.data) {
            this.normalFileList.splice(index, 1);
            this.deletedFileList.push(item);
          } else {
            this.$message.error("移入回收站失败");
          }
        });

      // this.normalFileList.splice(index, 1);
      // this.deletedFileList.push(item);
    },
    handleOpen(item) {
      this.$router.push({
        path: `/file/${item.title}`
      });
    },
    handleLog(item) {
      this.$router.push({
        path: `/log/${item.title}`
      });
    },
    handleDeleteTrue(item, index) {
      axios
        .post(this.deleteTrueFilePath, { filename: item.title })
        .then(response => {
          if (response.data) {
            this.deletedFileList.splice(index, 1);
          } else {
            this.$message.error("删除文件失败");
          }
        });
      // this.deletedFileList.splice(index, 1);
    },
    handleRecover(item, index) {
      axios.post(this.recoverPath, { filename: item.title }).then(response => {
        if (response.data) {
          this.deletedFileList.splice(index, 1);
          this.normalFileList.push(item);
        } else {
          this.$message.error("恢复文件失败");
        }
      });
      // this.deletedFileList.splice(index, 1);
      // this.normalFileList.push(item);
    },
    handleCancel(e) {
      this.visible = false;
    }
  }
};
</script>
<style>
#Recycled {
  background-color: #69b3b59c;
  width: auto;
  border-radius: 10px;
  padding: 15px;
  margin: 0 auto;
  box-shadow: 3px 3px 2px #dcdfdf;
}
.fileList-title {
  text-align: center;
  background-color: #69c8c3;
  width: 220px;
  border-radius: 40px;
  padding: 20px;
  margin: 0 auto;
  box-shadow: 3px 3px 2px #b7c4c4;
}
body {
  height: 100%;
  background-color: #edeeed;
}
</style>