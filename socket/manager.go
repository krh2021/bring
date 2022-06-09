package socket

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/tal-tech/go-zero/core/logx"
)

//客户端管理器
type ClientManager struct {
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
}

//客户端 Client
type Client struct {
	UserId int64
	Socket *websocket.Conn
}

const (
	NoticeType  = 1 //通知推送
	MessageType = 2 //IM消息
	ConnectType = 0 //连接消息
)

//创建客户端管理者
var Manager *ClientManager

func init() {
	s := new(ClientManager)
	s.Register = make(chan *Client)
	s.Unregister = make(chan *Client)
	s.Clients = make(map[*Client]bool)
	Manager = s
	logx.Info("client manager init...")
}

//Start
func (manager *ClientManager) Start() {
	logx.Info("client manager start ...")
	for {
		select {
		//如果有新的连接接入,就通过channel把连接传递给conn
		case conn := <-manager.Register:
			manager.Clients[conn] = true
			logx.Info(conn.UserId, " 连接成功...")
			go manager.heart(conn.UserId)
		case conn := <-manager.Unregister:
			if _, ok := manager.Clients[conn]; ok {
				conn.Socket.Close()
				logx.Info(conn.UserId, " 下线成功...")
				delete(manager.Clients, conn)
			}
		}
	}
}

//心跳响应
func (manager *ClientManager) heart(userId int64) {
	var client = new(Client)
	for k, v := range manager.Clients {
		if k.UserId == userId && v {
			client = k
		}
	}
	if client == nil {
		logx.Error(userId, "connet not get...")
		return
	}
	defer func() {
		manager.Unregister <- client
	}()
	logx.Info(userId, " 开启心跳检测...")
	client.Socket.SetReadLimit(8)
	client.Socket.SetReadDeadline(time.Now().Add(28800 * time.Second))
	for {
		messageType, message, err := client.Socket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Error(err)
			}
			break
		}
		logx.Stat(userId, " socket heart")
		client.Socket.WriteMessage(messageType, message)
	}
}

//将用户踢下线
func (manager *ClientManager) Offline(userId int64) {
	for conn := range manager.Clients {
		if conn.UserId == userId {
			manager.Unregister <- conn
		}
	}
}

//定义客户端管理的send方法
func (manager *ClientManager) Send(message interface{}, to int64) error {
	for conn := range manager.Clients {
		if conn.UserId == to {
			return conn.Socket.WriteJSON(message)
		}
	}
	return nil
}

//校验是否在线
func (manager *ClientManager) Status(uid int64) bool {
	for conn := range manager.Clients {
		if conn.UserId == uid {
			return true
		}
	}
	return false
}
