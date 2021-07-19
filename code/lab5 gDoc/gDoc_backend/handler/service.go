package handler

import (
	"context"
	"time"

	"gDoc_backend/db"
	"gDoc_backend/fs"
	"gDoc_backend/lock"
	"gDoc_backend/model"

	"github.com/astaxie/beego"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type _error struct {
	msg string
}

type FileRecord struct {
	Title     string `bson:"filename" json:"title"`
	IsDeleted bool   `bson:"deleteMark" json:"isDeleted"`
}

func (e *_error) Error() string {
	return e.msg
}

func LoadSheet(filename string) (model.Sheet, error) {
	return decodeFile(filename)
}

func ListFiles() ([]FileRecord, error) {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		cursor     *mongo.Cursor
		err        error
		result     []FileRecord
	)

	collection = client.Database("gDoc").Collection("files")
	filter := bson.M{}
	if cursor, err = collection.Find(context.TODO(), filter); err != nil {
		beego.Error("[MongoDB] Read data failed", err.Error())
	}
	defer func() {
		if err = cursor.Close(context.TODO()); err != nil {
			beego.Error("[MongoDB] Close cursor failed", err.Error())
		}
	}()

	if err = cursor.All(context.TODO(), &result); err != nil {
		beego.Error("[MongoDB] Parse cursor failed", err.Error())
	}

	return result, err
}

func CreateFile(fileName string) error {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		err        error
	)

	if err = initSheet(fileName); err != nil {
		beego.Error("[FS] Create file failed", err.Error())
		return err
	}

	collection = client.Database("gDoc").Collection("files")
	insert := bson.M{"filename": fileName, "deleteMark": false}
	filter := bson.M{"filename": fileName}
	count, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		beego.Error("[MongoDB] Count data failed", err.Error())
		return err
	}

	if count > 0 {
		return &_error{"duplicate file name"}
	}

	if _, err = collection.InsertOne(context.TODO(), insert); err != nil {
		beego.Error("[MongoDB] Insert data failed", err.Error())
	}

	return err
}

func DeleteFile(fileName string) error {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		err        error
	)

	collection = client.Database("gDoc").Collection("files")
	filter := bson.M{"filename": fileName}
	update := bson.M{"$set": bson.M{"deleteMark": true}}
	if _, err = collection.UpdateOne(context.TODO(), filter, update); err != nil {
		beego.Error("[MongoDB] Update data failed", err.Error())
	}

	return err
}

func DeleteForever(fileName string) error {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		err        error
	)

	collection = client.Database("gDoc").Collection("files")
	filter := bson.M{"filename": fileName}
	if _, err = collection.DeleteOne(context.TODO(), filter); err != nil {
		beego.Error("[MongoDB] Delete data failed", err.Error())
		return err
	}

	collection = client.Database("gDoc").Collection("log")
	if _, err = collection.DeleteMany(context.TODO(), filter); err != nil {
		beego.Error("[MongoDB] Delete data failed", err.Error())
	}

	lock.DeleteFileLock(fileName)

	return err

	// if err = fs.Delete(fileName); err != nil {
	// 	beego.Error("[FS] Delete data failed", err.Error())
	// }
	// return err
}

func Recycle(fileName string) error {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		err        error
	)

	collection = client.Database("gDoc").Collection("files")
	filter := bson.M{"filename": fileName}
	update := bson.M{"$set": bson.M{"deleteMark": false}}
	if _, err = collection.UpdateOne(context.TODO(), filter, update); err != nil {
		beego.Error("[MongoDB] Update data failed", err.Error())
	}

	return err
}

func GetLog(filename string) ([]model.Log, error) {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		cursor     *mongo.Cursor
		result     []model.Log
		err        error
	)

	collection = client.Database("gDoc").Collection("log")

	filter := bson.M{"filename": filename}
	if cursor, err = collection.Find(context.TODO(), filter); err != nil {
		beego.Error("[MongoDB] Read data failed", err.Error())
	}
	defer func() {
		if err = cursor.Close(context.TODO()); err != nil {
			beego.Error("[MongoDB] Close cursor failed", err.Error())
		}
	}()

	if err = cursor.All(context.TODO(), &result); err != nil {
		beego.Error("[MongoDB] Parse cursor failed", err.Error())
	}
	return result, err
}

func UndoLogs(filename string, timestamp time.Time) error {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		cursor     *mongo.Cursor
		result     []model.Log
		opts       options.FindOptions
		filePtr    *fs.File
		err        error
	)

	collection = client.Database("gDoc").Collection("log")
	filter := bson.M{"timestamp": bson.M{"$gt": timestamp}, "filename": filename}
	opts.Sort = bson.M{"timestamp": -1}
	if cursor, err = collection.Find(context.TODO(), filter, &opts); err != nil {
		beego.Error("[MongoDB] Read data failed", err.Error())
	}
	defer func() {
		if err = cursor.Close(context.TODO()); err != nil {
			beego.Error("[MongoDB] Close cursor failed", err.Error())
		}
	}()

	if err = cursor.All(context.TODO(), &result); err != nil {
		beego.Error("[MongoDB] Parse cursor failed", err.Error())
	}

	lock.LockFile(filename)
	defer lock.UnlockFile(filename)

	if filePtr, err = fs.Open(filename); err != nil {
		beego.Error("Open file error", err.Error())
		return err
	}
	defer filePtr.Close()

	for _, log := range result {
		if err = encodeCell(filePtr, log.Old); err != nil {
			beego.Error("Encode cell error", err.Error())
			return err
		}
	}

	if _, err = collection.DeleteMany(context.TODO(), filter); err != nil {
		beego.Error("[MongoDB] Delete data failed", err.Error())
	}

	return err
}

func logGridValue(filename, username string, new, old model.Cell) error {
	var (
		client     = db.GetMgoCli()
		collection *mongo.Collection
		insert     model.Log
		err        error
	)

	collection = client.Database("gDoc").Collection("log")
	insert = model.Log{
		File: filename,
		User: username,
		V:    new,
		Old:  old,
		Time: time.Now(),
	}
	if _, err = collection.InsertOne(context.TODO(), insert); err != nil {
		beego.Error("[MongoDB] Insert data failed", err.Error())
	}
	return err
}
