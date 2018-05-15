package main

import (
	"github.com/gin-gonic/gin"
	"github.com/SBOrg666/lite-yun-golang/utils"
	"github.com/gin-contrib/sessions"
	"github.com/satori/go.uuid"
	"github.com/jasonlvhit/gocron"
	"time"
)

func main() {
	router := gin.Default()
	store := sessions.NewCookieStore([]byte(uuid.Must(uuid.NewV4()).String()))
	store.Options(sessions.Options{MaxAge: 0, HttpOnly: false})
	router.Use(sessions.Sessions("session", store))

	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")
	router.LoadHTMLFiles("./template/index.html",
		"./template/login.html",
		"./template/processes.html",
		"./template/path.html",
		"./template/about.html",
		"./template/authors.html",
	)

	router.GET("/", utils.CheckLoginIn(), utils.IndexHandler_get)

	LoginGroup := router.Group("/")
	{
		LoginGroup.GET("/login", utils.LoginHandler_get)
		LoginGroup.POST("/login", utils.LoginHandler_post)
	}

	router.GET("/Systeminfo", utils.CheckLoginIn(), func(c *gin.Context) {
		utils.SystemInfoHandler_ws(c.Writer, c.Request)
	})

	router.GET("/processes.html", utils.CheckLoginIn(), utils.ProcessHandler_get)

	router.GET("/processesInfo", utils.CheckLoginIn(), func(c *gin.Context) {
		utils.ProcessInfoHandler_ws(c.Writer, c.Request)
	})

	router.GET("/path", utils.CheckLoginIn(), utils.PathHandler_get)

	DownloadGroup := router.Group("/")
	{
		DownloadGroup.GET("/download", utils.CheckLoginIn(), utils.DownloadHandler_get)
		DownloadGroup.POST("/download", utils.CheckLoginIn(), utils.DownloadHandler_post)
	}

	router.POST("/upload", utils.CheckLoginIn(), utils.UploadHandler_post)

	router.POST("/delete", utils.CheckLoginIn(), utils.DeleteHandler_post)

	AboutGroup := router.Group("/")
	{
		AboutGroup.GET("/about", utils.CheckLoginIn(), utils.AboutHandler_get)
		AboutGroup.GET("/authors", utils.CheckLoginIn(), utils.AuthorsHandler_get)
	}

	utils.Upload_data = make([]uint64, 5)
	utils.Download_data = make([]uint64, 5)
	utils.InitUpload = 0
	utils.InitDownload = 0
	utils.Current_Month = int(time.Now().Month())
	gocron.Every(1).Day().Do(utils.UpdateNetworkData)

	router.Run(":8000")
}
