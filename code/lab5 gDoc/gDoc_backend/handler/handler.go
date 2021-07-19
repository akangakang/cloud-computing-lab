package handler

import (
	"encoding/json"
	"gDoc_backend/model"

	"github.com/astaxie/beego"
)

type rspType int

const (
	RSP_SUCCESS rspType = 0
	RSP_SELF    rspType = 1
	RSP_OTHER   rspType = 2
	RSP_REGION  rspType = 3
	RSP_FAIL    rspType = 999
)

type Response struct {
	Type     rspType `json:"type"`
	UserName string  `json:"username"`
	Id       string  `json:"id"`
	Data     string  `json:"data"`
}

type handler func(reqmsg []byte, filename, username string)

var handlers map[string]handler

func InitHandlers() {
	handlers = map[string]handler{
		"v":   updateGrid,
		"rv":  updateGridMulti,
		"cg":  updateGridConfig,
		"all": updateGridCommon,
		"fc":  updateCalcChain,
		"drc": updateRowColumn,
		"arc": updateRowColumn,
		// "fsc":  updateFilter,
		// "fsr":  updateFilter,
		// "sha":  addSheet,
		"shc":  copySheet,
		"shd":  deleteSheet,
		"shre": recoverSheet,
		"shr":  updateSheetOrder,
		"shs":  toggleSheet,
		"sh":   hideOrShowSheet,
	}
}

func HandleMsg(reqmsg []byte, name string, filename string) []byte {
	var msg struct {
		T string `json:"t"`
	}
	rsp := Response{}
	json.Unmarshal(reqmsg, &msg)
	rsp.Id = name
	rsp.UserName = name
	switch msg.T {
	case "v", "rv", "rv_end", "cg", "all", "fc", "drc", "arc", "f", "fsc", "fsr", "sha", "shc", "shd", "shr", "shre", "sh", "c", "na":
		rsp.Type = RSP_OTHER
	case "mv":
		rsp.Type = RSP_REGION
		// rsp.Id = uid
		// rsp.UserName = s.GetName()
	// case "": //离线情况下把更新指令打包批量下发给客户端
	// 	rsp.Type = 4
	default:
		rsp.Type = RSP_SELF
	}

	handler, ok := handlers[msg.T]
	if ok {
		handler(reqmsg, filename, name)
	}
	rsp.Data = string(reqmsg)
	jsonBytes, _ := json.Marshal(rsp)
	return jsonBytes
}

func updateGrid(reqmsg []byte, filename, username string) {
	req := new(model.UpdateV)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}

	persistGridValue(req, filename, username)
	// logGridValue(filename, username, req)
	// ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	// defer cancel()
	// err = s.d.UpdateGridValue(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "update grid failed")
	// }
}

func updateGridMulti(reqmsg []byte, filename, username string) {
	req := new(model.UpdateRV)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}

	persistGridMulti(req, filename)
	// if len(req.Range.Column) < 2 || len(req.Range.Row) < 2 {
	// 	log.Errorw(ctx, "req", req, "msg", "invalid params")
	// 	return
	// }
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.UpdateGridMulti(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "update grid failed")
	// }
}

func updateGridConfig(reqmsg []byte, filename, username string) {
	req := new(model.UpdateCG)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}

	persistGridConfig(req, filename)
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.UpdateGridConfig(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "update grid failed")
	// }
}

func updateGridCommon(reqmsg []byte, filename, username string) {
	req := new(model.UpdateCommon)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	persistGridCommon(req, filename)
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.UpdateGridCommon(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "update grid failed")
	// }
}

func updateCalcChain(reqmsg []byte, filename, username string) {
	req := new(model.UpdateCalcChain)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	persistCalcChain(req, filename)
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.UpdateCalcChain(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "update calc chain failed")
	// }
}

func updateRowColumn(reqmsg []byte, filename, username string) {
	req := new(model.UpdateRowColumn)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.UpdateRowColumn(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "update calc chain failed")
	// }
}

// func updateFilter(reqmsg []byte) {
// 	req := new(UpdateFilter)
// 	err := json.Unmarshal(reqmsg, req)
// 	if err != nil {
// 		log.Errorw(ctx, "err", err, "jsonstr", string(reqmsg), "msg", "json unmarshal error")
// 		return
// 	}
// 	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
// 	defer cancel()
// 	err = s.d.UpdateFilter(ctx, s.gridKey, req)
// 	if err != nil {
// 		log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "update calc chain failed")
// 	}
// }

// func addSheet(reqmsg []byte) {
// 	req := new(AddSheet)
// 	err := json.Unmarshal(reqmsg, req)
// 	if err != nil {
// 		log.Errorw(ctx, "err", err, "jsonstr", string(reqmsg), "msg", "json unmarshal error")
// 		return
// 	}
// 	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
// 	defer cancel()
// 	err = s.d.AddSheet(ctx, s.gridKey, req)
// 	if err != nil {
// 		log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "add sheet failed")
// 	}
// }

func copySheet(reqmsg []byte, filename, username string) {
	req := new(model.CopySheet)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.CopySheet(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "add sheet failed")
	// }
}

func deleteSheet(reqmsg []byte, filename, username string) {
	req := new(model.DeleteSheet)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	// ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	// defer cancel()
	// err = s.d.DeleteSheet(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "add sheet failed")
	// }
}

func recoverSheet(reqmsg []byte, filename, username string) {
	req := new(model.RecoverSheet)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.RecoverSheet(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "add sheet failed")
	// }
}

func updateSheetOrder(reqmsg []byte, filename, username string) {
	req := new(model.UpdateSheetOrder)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.UpdateSheetOrder(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "add sheet failed")
	// }
}

func toggleSheet(reqmsg []byte, filename, username string) {
	req := new(model.ToggleSheet)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.ToggleSheet(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "add sheet failed")
	// }
}

func hideOrShowSheet(reqmsg []byte, filename, username string) {
	req := new(model.HideOrShowSheet)
	err := json.Unmarshal(reqmsg, req)
	if err != nil {
		beego.Error("Json unmarshal error")
		return
	}
	// ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	// defer cancel()
	// err = s.d.HideOrShowSheet(ctx, s.gridKey, req)
	// if err != nil {
	// 	log.Errorw(ctx, "err", err, "gridKey", s.gridKey, "req", req, "msg", "add sheet failed")
	// }
}
