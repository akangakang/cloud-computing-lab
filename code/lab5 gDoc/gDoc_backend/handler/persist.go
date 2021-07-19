package handler

import (
	"encoding/json"
	"gDoc_backend/fs"
	"gDoc_backend/lock"
	"gDoc_backend/model"
	"io"

	"github.com/astaxie/beego"
)

func decodeFile(filename string) (sheet model.Sheet, err error) {
	filePtr, err := fs.Open(filename)
	if err != nil { // assume file doesn't exist
		beego.Debug("File doesn't exist", filename, err.Error())
		filePtr, err := fs.Create(filename)
		if err != nil {
			beego.Error("Create file failed", err.Error())
			return sheet, err
		}

		lock.AddFileLock(filename)

		sheet = model.Sheet{
			Name:   filename,
			Order:  0,
			Index:  "index_" + filename,
			Status: 1,
		}
		buffer, err := json.Marshal(sheet)
		if err != nil {
			beego.Error("Marshal error", err.Error())
			return sheet, err
		}
		lock.LockMeta(filename)
		_, err = filePtr.WriteMeta(buffer)
		lock.UnlockMeta(filename)
		if err != nil {
			beego.Error("Write file error", err.Error())
			return sheet, err
		}
		sheet.Status = 0
		return sheet, nil
	}

	lock.LockFile(filename)
	buffer, n, err := filePtr.Read()
	lock.UnlockFile(filename)

	if err != nil && err != io.EOF && err != fs.ErrNotExist {
		beego.Error("Readfile error:", err.Error())
		return sheet, err
	}
	beego.Debug("Read file content: ", string(buffer[:n]))
	err = json.Unmarshal(buffer[:n], &sheet)
	if err != nil {
		beego.Error("Unmarshal error", err.Error())
	}
	return sheet, err
}

func initSheet(filename string) (err error) {
	var (
		filePtr *fs.File
		sheet   model.Sheet
		buffer  []byte
	)

	if filePtr, err = fs.Create(filename); err != nil {
		beego.Error("[FS] Create file failed", err.Error())
		return err
	}

	lock.AddFileLock(filename)

	sheet = model.Sheet{
		Name:   filename,
		Order:  0,
		Index:  "index_" + filename,
		Status: 1,
	}
	if buffer, err = json.Marshal(sheet); err != nil {
		beego.Error("Marshal error", err.Error())
		return err
	}

	lock.LockMeta(filename)
	defer lock.UnlockMeta(filename)

	if _, err = filePtr.WriteMeta(buffer); err != nil {
		beego.Error("Write file error", err.Error())
		return err
	}

	return nil
}

func decodeCell(filePtr *fs.File, row uint32, col uint32) (model.Cell, error) {
	buffer := make([]byte, 2048)

	length, err := filePtr.ReadAt(buffer, row, col)
	if err != nil {
		if err != fs.ErrNotExist {
			beego.Error("Read cell data error", err.Error())
		} else {
			err = nil
		}
		return model.Cell{}, err
	}

	cell := model.Cell{}
	err = json.Unmarshal(buffer[:length], &cell)
	if err != nil {
		beego.Error("Unmarshal error", err.Error())
	}
	return cell, err
}

func encodeCell(filePtr *fs.File, cell model.Cell) error {
	buffer, err := json.Marshal(cell)
	if err != nil {
		beego.Error("Marshal error", err.Error())
		return err
	}

	_, err = filePtr.WriteAt(buffer, uint32(cell.R), uint32(cell.C), " ")
	if err != nil {
		beego.Error("Write cell data error", err.Error())
	}
	return err
}

func decodeCommon(filePtr *fs.File) (model.Sheet, error) {
	buffer := make([]byte, 8194)
	_, err := filePtr.ReadMeta(buffer)
	if err != nil {
		if err != fs.ErrNotExist {
			beego.Error("Read common data error", err.Error())
		} else {
			err = nil
		}
		return model.Sheet{}, err
	}
	beego.Debug("Decode meta:", string(buffer))
	common := model.Sheet{}
	err = json.Unmarshal(buffer, &common)
	if err != nil {
		beego.Error("Unmarshal error", err.Error())
	}
	return common, err
}

func encodeCommon(filePtr *fs.File, common model.Sheet) error {
	buffer, err := json.Marshal(common)
	if err != nil {
		beego.Error("Marshal error", err.Error())
		return err
	}

	_, err = filePtr.WriteMeta(buffer)
	if err != nil {
		beego.Error("Write common data error", err.Error())
	}
	return err
}

func persistGridValue(req *model.UpdateV, filename, username string) {
	filePtr, err := fs.Open(filename)
	if err != nil {
		beego.Error("Open file error", err.Error())
		return
	}
	defer filePtr.Close()

	lock.LockCell(filename, uint32(req.Cell.R), uint32(req.Cell.C))
	defer lock.UnlockCell(filename, uint32(req.Cell.R), uint32(req.Cell.C))

	cell, err := decodeCell(filePtr, uint32(req.Cell.R), uint32(req.Cell.C))
	if err != nil {
		beego.Error("Decode cell error", err.Error())
		cell = model.Cell{}
	}

	cell.R = req.Cell.R
	cell.C = req.Cell.C
	logGridValue(filename, username, req.Cell, cell)
	cell.V = req.Cell.V

	err = encodeCell(filePtr, cell)
	if err != nil {
		beego.Error("Encode cell error", err.Error())
	}
}

func persistGridMulti(req *model.UpdateRV, filename string) {
	filePtr, err := fs.Open(filename)
	if err != nil {
		beego.Error("Open file error", err.Error())
		return
	}
	defer filePtr.Close()

	for i, row := 0, req.Range.Row[0]; row <= req.Range.Row[1]; i, row = i+1, row+1 {
		for j, col := 0, req.Range.Column[0]; col <= req.Range.Column[1]; j, col = j+1, col+1 {
			cell_row, cell_col := req.Range.Row[0]+i, req.Range.Column[0]+j

			lock.LockCell(filename, uint32(cell_row), uint32(cell_col))

			// cell, err := decodeCell(filePtr, uint32(cell_row), uint32(cell_col))
			// if err != nil {
			// 	beego.Error("Decode cell error", err.Error())
			// }
			cell := model.Cell{}

			cell.V = req.V[i][j]
			cell.R = model.FlexInt(cell_row)
			cell.C = model.FlexInt(cell_col)

			err = encodeCell(filePtr, cell)
			if err != nil {
				beego.Error("Encode cell error", err.Error())
			}

			lock.UnlockCell(filename, uint32(cell_row), uint32(cell_col))
		}
	}
}

func persistGridConfig(req *model.UpdateCG, filename string) {
	filePtr, err := fs.Open(filename)
	if err != nil {
		beego.Error("Open file error", err.Error())
		return
	}
	defer filePtr.Close()

	lock.LockMeta(filename)
	defer lock.UnlockMeta(filename)

	sheet, err := decodeCommon(filePtr)
	if err != nil {
		beego.Error("Decode common error", err.Error())
	}

	convert, err := json.Marshal(req.V)
	if err != nil {
		beego.Error("Marshal error", err.Error())
		return
	}

	if sheet.Config == nil {
		sheet.Config = new(model.SheetConfig)
	}

	switch req.K {
	case "borderInfo":
		var borderInfo []model.Border
		err = json.Unmarshal(convert, &borderInfo)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Config.BorderInfo = borderInfo
	case "rowhidden":
		var rowhidden map[string]float64
		err = json.Unmarshal(convert, &rowhidden)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Config.Rowhidden = rowhidden
	case "colhidden":
		var colhidden map[string]float64
		err = json.Unmarshal(convert, &colhidden)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Config.Colhidden = colhidden
	case "rowlen":
		var rowlen map[string]float64
		err = json.Unmarshal(convert, &rowlen)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Config.Rowlen = rowlen
	case "columnlen":
		var columnlen map[string]float64
		err = json.Unmarshal(convert, &columnlen)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Config.Columnlen = columnlen
	}

	err = encodeCommon(filePtr, sheet)
	if err != nil {
		beego.Error("Encode common error", err.Error())
	}
}

func persistGridCommon(req *model.UpdateCommon, filename string) {
	filePtr, err := fs.Open(filename)
	if err != nil {
		beego.Error("Open file error", err.Error())
		return
	}
	defer filePtr.Close()

	lock.LockMeta(filename)
	defer lock.UnlockMeta(filename)

	sheet, err := decodeCommon(filePtr)
	if err != nil {
		beego.Error("Decode common error", err.Error())
	}

	convert, err := json.Marshal(req.V)
	if err != nil {
		beego.Error("Marshal error", err.Error())
		return
	}
	switch req.K {
	case "fozen":
		var frozen *model.Frozen = new(model.Frozen)
		err = json.Unmarshal(convert, frozen)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Frozen = frozen
	case "name":
		sheet.Name = req.V.(string)
	case "color":
		sheet.Color = req.V.(string)
	case "config":
		var config *model.SheetConfig = new(model.SheetConfig)
		err = json.Unmarshal(convert, config)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Config = config
	case "filter_select":
		var filterSelect *model.FilterSelect = new(model.FilterSelect)
		err = json.Unmarshal(convert, filterSelect)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.FilterSelect = filterSelect
	case "filter":
		var filter map[string]model.Filter
		err = json.Unmarshal(convert, &filter)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Filter = filter
	case "luckysheet_alternateformat_save":
		var alternateFormatSave []model.AlternateFormatSave
		err = json.Unmarshal(convert, &alternateFormatSave)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.AlternateFormatSave = alternateFormatSave
	case "luckysheet_conditionformat_save":
		var conditionFormatSave []model.ConditionFormatSave
		err = json.Unmarshal(convert, &conditionFormatSave)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.ConditionFormatSave = conditionFormatSave
	case "pivotTable":
		var pivotTable *model.PivotTable = new(model.PivotTable)
		err = json.Unmarshal(convert, pivotTable)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.PivotTable = pivotTable
	case "dynamicArray":
		var dynamicArray []model.DynamicArray
		err = json.Unmarshal(convert, &dynamicArray)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.DynamicArray = dynamicArray
	case "images":
		var images interface{}
		err = json.Unmarshal(convert, &images)
		if err != nil {
			beego.Error("Unmarshal error", err.Error())
			return
		}
		sheet.Images = images
	}

	err = encodeCommon(filePtr, sheet)
	if err != nil {
		beego.Error("Encode common error", err.Error())
	}
}

func persistCalcChain(req *model.UpdateCalcChain, filename string) {
	filePtr, err := fs.Open(filename)
	if err != nil {
		beego.Error("Open file error", err.Error())
		return
	}
	defer filePtr.Close()

	lock.LockMeta(filename)
	defer lock.UnlockMeta(filename)

	sheet, err := decodeCommon(filePtr)
	if err != nil {
		beego.Error("Decode common error", err.Error())
		return
	}

	var calcChain model.CalcChain
	err = json.Unmarshal([]byte(req.V), &calcChain)
	if err != nil {
		beego.Error("Unmarshal error", err.Error())
		return
	}
	switch req.Op {
	case "add":
		sheet.CalcChain = append(sheet.CalcChain, calcChain)
	case "update":
		sheet.CalcChain[req.Pos] = calcChain
	case "del":
		sheet.CalcChain = append(sheet.CalcChain[:req.Pos], sheet.CalcChain[req.Pos+1:]...)
	}

	err = encodeCommon(filePtr, sheet)
	if err != nil {
		beego.Error("Encode common error", err.Error())
	}
}
