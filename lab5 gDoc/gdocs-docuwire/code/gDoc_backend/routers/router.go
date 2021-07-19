package routers

import (
	"gDoc_backend/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/connect", &controllers.ServerController{}, "get,post:Connect")
	beego.Router("/load", &controllers.ServerController{}, "post:Load")
	beego.Router("/getFileList", &controllers.ServerController{}, "get:List")
	beego.Router("/createFile", &controllers.ServerController{}, "post:Create")
	beego.Router("/deleteFile", &controllers.ServerController{}, "post:Delete")
	beego.Router("/deleteRecycled", &controllers.ServerController{}, "post:DeleteForever")
	beego.Router("/recoverFile", &controllers.ServerController{}, "post:Recycle")
	beego.Router("/getLog", &controllers.ServerController{}, "get:GetLog")
	beego.Router("/rollback", &controllers.ServerController{}, "post:Rollback")
}
