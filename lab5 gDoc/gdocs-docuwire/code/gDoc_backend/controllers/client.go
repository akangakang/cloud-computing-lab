package controllers

import (
	"encoding/json"
	"gDoc_backend/handler"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
	"github.com/samuel/go-zookeeper/zk"
)

var zKconnect *zk.Conn

type Client struct {
	conn *websocket.Conn
	name string
}

type Message struct {
	EventType byte   `json:"type"`
	Name      string `json:"name"`
	Message   []byte `json:"message"`
}

var (
	join    = make(chan Client, 10)
	leave   = make(chan Client, 10)  // 用户退出通道
	message = make(chan Message, 10) // 消息通道
	clients = make(map[Client]bool)  // 用户映射
)

func init() {

	go broadcaster()
}

func broadcaster() {
	for {

		select {
		case msg := <-message:
			// str := fmt.Sprintf("broadcaster-----------%s send message: %s\n", msg.Name, msg.Message)
			// beego.Info(str)
			for client := range clients {
				if client.conn.WriteMessage(websocket.TextMessage, msg.Message) != nil {
					beego.Error("Fail to write message")
				}
			}

		case client := <-join:
			beego.Info("Client join:", client.name)
			clients[client] = true

			rsp := handler.Response{
				Type:     handler.RSP_SUCCESS,
				UserName: client.name,
				Id:       client.name,
			}
			jsonBytes, _ := json.Marshal(rsp)
			client.conn.WriteMessage(websocket.TextMessage, jsonBytes)

			// var msg Message
			// msg.Name = client.name
			// msg.EventType = 1
			// msg.Message = fmt.Sprintf("%s join in, there are %d preson in room", client.name, len(clients))

			// message <- msg

		case client := <-leave:
			beego.Info("Client leave:", client.name)

			if !clients[client] {
				break
			}
			delete(clients, client)
		}
	}
}
