package model

import (
	"time"
)

type UpdateV struct {
	Cell
	I string `json:"i" bson:"i"`
	T string `json:"t" bson:"t"`
}

type UpdateRV struct {
	I     string `json:"i" bson:"i"`
	Range struct {
		Column []int `json:"column" bson:"column"`
		Row    []int `json:"row" bson:"row"`
	} `json:"range" bson:"range"`
	T string        `json:"t" bson:"t"`
	V [][]CellValue `json:"v" bson:"v"`
}

type UpdateCG struct {
	T string      `json:"t" bson:"t"`
	I string      `json:"i" bson:"i"`
	K string      `json:"k" bson:"k"`
	V interface{} `json:"v" bson:"v"`
}

type UpdateCommon struct {
	T string      `json:"t" bson:"t"`
	I string      `json:"i" bson:"i"`
	K string      `json:"k" bson:"k"`
	V interface{} `json:"v" bson:"v"`
}

type UpdateCalcChain struct {
	I   string `json:"i" bson:"i"`
	Op  string `json:"op" bson:"op"`
	Pos int    `json:"pos" bson:"pos"`
	T   string `json:"t" bson:"t"`
	V   string `json:"v" bson:"v"`
}

type UpdateRowColumn struct {
	I string `json:"i"`
	T string `json:"t"`
	V struct {
		Index int `json:"index"`
		Len   int `json:"len"`
	} `json:"v"`
	RC string `json:"rc"`
}

type UpdateFilter struct {
	I string       `json:"i"`
	T string       `json:"t"`
	V *FilterValue `json:"v"`
}

type FilterValue struct {
	Filter       []Filter     `json:"filter"`
	FilterSelect FilterSelect `json:"filter_select"`
}

type AddSheet struct {
	I string `json:"i"`
	T string `json:"t"`
	V *Sheet `json:"v"`
}

type CopySheet struct {
	I string `json:"i"`
	T string `json:"t"`
	V struct {
		CopyIndex string `json:"copyindex"`
		Name      string `json:"name"`
	} `json:"v"`
}

type DeleteSheet struct {
	I string `json:"i"`
	T string `json:"t"`
	V struct {
		DeleteIndex string `json:"deleIndex"`
	} `json:"v"`
}

type RecoverSheet struct {
	I string `json:"i"`
	T string `json:"t"`
	V struct {
		RecoverIndex string `json:"reIndex"`
	} `json:"v"`
}

type UpdateSheetOrder struct {
	I string         `json:"i"`
	T string         `json:"t"`
	V map[string]int `json:"v"`
}

type ToggleSheet struct {
	I string `json:"i"`
	T string `json:"t"`
	V string `json:"v"`
}

type HideOrShowSheet struct {
	I   string `json:"i"`
	T   string `json:"t"`
	V   int    `json:"v"`
	Op  string `json:"op"`
	Cur string `json:"cur"`
}

type Log struct {
	File string    `json:"filename" bson:"filename"`
	User string    `json:"user" bson:"user"`
	V    Cell      `json:"value" bson:"value"`
	Old  Cell      `json:"oldvalue" bson:"oldvalue"`
	Time time.Time `json:"timestamp" bson:"timestamp"`
}
