package controllers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"gDoc_backend/handler"
	"gDoc_backend/model"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
	"golang.org/x/text/encoding/charmap"
)

type ServerController struct {
	beego.Controller
}

type FileRequest struct {
	Filename string `json:"filename"`
}

type RollbackRequest struct {
	Filename  string    `json:"filename"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (c *ServerController) Load() {
	filename := c.Ctx.Input.Query("filename")
	if len(filename) == 0 {
		beego.Error("filename is NULL")
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	sheet, err := handler.LoadSheet(filename)
	if err != nil {
		beego.Error("Load sheet failed", err.Error())
	}

	var sheets []model.Sheet
	sheets = append(sheets, sheet)

	jsonString, err := json.Marshal(sheets)
	if err != nil {
		beego.Error("Json marshall error")
		return
	}
	c.Data["json"] = string(jsonString)
	c.ServeJSON()
}

func (c *ServerController) List() {
	var (
		fileList []handler.FileRecord
		err      error
	)

	if fileList, err = handler.ListFiles(); err != nil {
		beego.Error("List file error", err.Error())
		c.Ctx.ResponseWriter.WriteHeader(500)
		return
	}

	c.Data["json"] = fileList
	c.ServeJSON()
}

func (c *ServerController) Create() {
	var (
		result   bool
		request  FileRequest
		fileName string
		err      error
	)

	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &request); err != nil {
		beego.Error("Json unmarshal error", err.Error())
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}
	fileName = request.Filename
	if fileName == "" {
		beego.Error("Invalid file name:", fileName)
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	err = handler.CreateFile(fileName)
	if err != nil {
		beego.Error("Create file failed", err.Error())
		result = false
	} else {
		result = true
	}

	c.Data["json"] = result
	c.ServeJSON()
}

func (c *ServerController) Delete() {
	var (
		result   bool
		request  FileRequest
		fileName string
		err      error
	)

	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &request); err != nil {
		beego.Error("Json unmarshal error", err.Error())
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}
	fileName = request.Filename
	if fileName == "" {
		beego.Error("Invalid file name:", fileName)
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	if err := handler.DeleteFile(fileName); err != nil {
		beego.Error("Delete file failed", err.Error())
		result = false
	} else {
		result = true
	}

	c.Data["json"] = result
	c.ServeJSON()
}

func (c *ServerController) DeleteForever() {
	var (
		result   bool
		request  FileRequest
		fileName string
		err      error
	)

	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &request); err != nil {
		beego.Error("Json unmarshal error", err.Error())
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}
	fileName = request.Filename
	if fileName == "" {
		beego.Error("Invalid file name:", fileName)
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	if err := handler.DeleteForever(fileName); err != nil {
		beego.Error("Delete file failed", err.Error())
		result = false
	} else {
		result = true
	}

	c.Data["json"] = result
	c.ServeJSON()
}

func (c *ServerController) Recycle() {
	var (
		result   bool
		request  FileRequest
		fileName string
		err      error
	)

	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &request); err != nil {
		beego.Error("Json unmarshal error", err.Error())
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}
	fileName = request.Filename
	if fileName == "" {
		beego.Error("Invalid file name:", fileName)
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	if err := handler.Recycle(fileName); err != nil {
		beego.Error("Delete file failed", err.Error())
		result = false
	} else {
		result = true
	}

	c.Data["json"] = result
	c.ServeJSON()
}

func (c *ServerController) GetLog() {
	var (
		fileName string
		rep      []model.Log
		err      error
	)

	fileName = c.Ctx.Input.Query("filename")
	if fileName == "" {
		beego.Error("Invalid file name:", fileName)
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	if rep, err = handler.GetLog(fileName); err != nil {
		beego.Error("Get log failed", err.Error())
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	c.Data["json"] = rep
	c.ServeJSON()
}

func (c *ServerController) Rollback() {
	var (
		result  bool
		request RollbackRequest
		err     error
	)
	if err = json.Unmarshal(c.Ctx.Input.RequestBody, &request); err != nil {
		beego.Error("Json unmarshal error", err.Error())
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	if request.Filename == "" {
		beego.Error("Invalid file name:", request.Filename)
		c.Ctx.ResponseWriter.WriteHeader(400)
		return
	}

	if err = handler.UndoLogs(request.Filename, request.Timestamp); err != nil {
		beego.Error("Undo logs failed", err.Error())
		result = false
	} else {
		result = true
	}

	c.Data["json"] = result
	c.ServeJSON()
}

func (c *ServerController) Connect() {
	name := c.Ctx.Input.Query("name")
	if len(name) == 0 {
		beego.Error("name is NULL")
		c.Redirect("/", 302)
		return
	}
	filename := c.Ctx.Input.Query("filename")
	if len(filename) == 0 {
		beego.Error("filename is NULL")
		c.Redirect("/", 302)
		return
	}
	// 检验http头中upgrader属性，若为websocket，则将http协议升级为websocket协议
	var upgrader = websocket.Upgrader{
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)

	if _, ok := err.(websocket.HandshakeError); ok {
		beego.Error("Not a websocket connection")
		http.Error(c.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		beego.Error("Cannot setup WebSocket connection:", err)
		return
	}

	var client Client
	client.name = name
	client.conn = conn

	// 如果用户列表中没有该用户
	if !clients[client] {
		join <- client
		beego.Info("user:", client.name, "websocket connect success!")
	}

	// 当函数返回时，将该用户加入退出通道，并断开用户连接
	defer func() {
		leave <- client
		client.conn.Close()
	}()

	// 由于WebSocket一旦连接，便可以保持长时间通讯，则该接口函数可以一直运行下去，直到连接断开
	for {
		// 读取消息。如果连接断开，则会返回错误
		_, msgStr, err := client.conn.ReadMessage()

		// 如果返回错误，就退出循环
		if err != nil {
			break
		}

		reqmsg, err := ungzip(msgStr)
		if err != nil {
			beego.Error("Ungzip error")
			continue
		}
		reqmsg = bytes.TrimSpace(bytes.Replace(reqmsg, newline, space, -1))
		// beego.Info("WS receive: " + string(reqmsg))
		response := handler.HandleMsg(reqmsg, name, filename)

		//如果没有错误，则把用户发送的信息放入message通道中
		var msg Message
		msg.Name = client.name
		msg.EventType = 0
		msg.Message = response
		message <- msg
	}
}

func ungzip(gzipmsg []byte) (reqmsg []byte, err error) {
	if len(gzipmsg) == 0 {
		return
	}
	if string(gzipmsg) == "rub" {
		reqmsg = gzipmsg
		return
	}
	e := charmap.ISO8859_1.NewEncoder()
	encodeMsg, err := e.Bytes(gzipmsg)
	if err != nil {
		return
	}
	b := bytes.NewReader(encodeMsg)
	r, err := gzip.NewReader(b)
	if err != nil {
		return
	}
	defer r.Close()
	reqmsg, err = ioutil.ReadAll(r)
	if err != nil {
		return
	}
	reqstr, err := url.QueryUnescape(string(reqmsg))
	if err != nil {
		return
	}
	reqmsg = []byte(reqstr)
	return
}
