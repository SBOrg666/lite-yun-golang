package utils

import (
	"database/sql"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"time"
	"strings"
	"github.com/shirou/gopsutil/process"
	"strconv"
	"os"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"github.com/satori/go.uuid"
	"io"
	"log"
)

type User struct {
	Name     string `form:"username"`
	Password string `form:"password"`
}

type FileList struct {
	Files []string `json:"files"`
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
		panic(err)
	}
}

func logErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func IndexHandler_get(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func LoginHandler_get(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func LoginHandler_post(c *gin.Context) {
	db, err := sql.Open("sqlite3", "./ACCOUNT.sqlite")
	checkErr(err)
	var user User
	err = c.ShouldBind(&user)
	checkErr(err)
	var passwordInDb string
	rows, err := db.Query(fmt.Sprintf("SELECT PASSWORD FROM USER WHERE NAME = %q", user.Name))
	for rows.Next() {
		err = rows.Scan(&passwordInDb)
		checkErr(err)
		break
	}
	rows.Close()
	db.Close()
	if user.Password != passwordInDb {
		c.String(http.StatusOK, "failed")
	} else {
		session := sessions.Default(c)
		session.Set("login", "true")
		session.Save()
		c.String(http.StatusOK, "ok")
	}
}

var wsupgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

func SystemInfoHandler_ws(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	checkErr(err)
	ticker := time.NewTicker(time.Second * 3)
	defer func() {
		ticker.Stop()
	}()
	err = conn.WriteJSON(gin.H{"log_info": GetLog_Info(Logfile),
		"cpu_info": GetCpu_Info(),
		"sys_info": GetSys_Info(),
		"mem_info": GetMem_Info(),
		"swap_info": GetSwap_Info(),
		"disk_info": GetDisk_Info(),
		"network_info": GetNetwork_Info(),
	})
	if err != nil {
		//log.Println("websocket disconnect")
		return
	}
	for range ticker.C {
		//log.Println("websocket ok")
		err := conn.WriteJSON(gin.H{"log_info": GetLog_Info(Logfile),
			"cpu_info": GetCpu_Info(),
			"sys_info": GetSys_Info(),
			"mem_info": GetMem_Info(),
			"swap_info": GetSwap_Info(),
			"disk_info": GetDisk_Info(),
			"network_info": GetNetwork_Info(),
		})
		if err != nil {
			//log.Println("websocket disconnect")
			break
		}
	}
}

var wsupgrader2 = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

func ProcessInfoHandler_ws(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader2.Upgrade(w, r, nil)
	checkErr(err)
	ticker := time.NewTicker(time.Second * 3)
	defer func() {
		ticker.Stop()
	}()

	info, err := GetProcess_Info()
	if err == nil {
		err := conn.WriteJSON(gin.H{
			"ProcessInfo": info,
		})
		if err != nil {
			//log.Println("websocket disconnect")
			conn.Close()
			return
		}
	}

	go func() {
		for {
			t, msg, err := conn.ReadMessage()
			//log.Println(err)
			if err == nil {
				s := string(msg[:])
				info := strings.Split(s, " ")
				pid, err := strconv.Atoi(info[0])
				if err == nil {
					if b, err := process.PidExists(int32(pid)); b && err == nil {
						pro, err := process.NewProcess(int32(pid))
						if err == nil {
							create_time, err := pro.CreateTime()
							if err == nil && fmt.Sprint(create_time) == info[2] {
								if info[1] == "1" {
									err = pro.Suspend()
									if err == nil {
										conn.WriteMessage(t, []byte(info[0]+" succeed"))
									} else {
										conn.WriteMessage(t, []byte(fmt.Sprint(err)))
									}
								} else if info[1] == "2" {
									err = pro.Resume()
									if err == nil {
										conn.WriteMessage(t, []byte(info[0]+" succeed"))
									} else {
										conn.WriteMessage(t, []byte(fmt.Sprint(err)))
									}
								} else if info[1] == "3" {
									err = pro.Terminate()
									if err == nil {
										conn.WriteMessage(t, []byte(info[0]+" succeed"))
									} else {
										conn.WriteMessage(t, []byte(fmt.Sprint(err)))
									}
								} else if info[1] == "4" {
									err = pro.Kill()
									if err == nil {
										conn.WriteMessage(t, []byte(info[0]+" succeed"))
									} else {
										conn.WriteMessage(t, []byte(fmt.Sprint(err)))
									}
								} else {
									conn.WriteMessage(t, []byte("invalid operation"))
								}
							} else {
								conn.WriteMessage(t, []byte("create time not match"))
							}
						} else {
							conn.WriteMessage(t, []byte(fmt.Sprint(err)))
						}
					} else {
						conn.WriteMessage(t, []byte("process not exist"))
					}
				} else {
					conn.WriteMessage(t, []byte(fmt.Sprint(err)))
				}
			} else {
				if fmt.Sprint(err) == "websocket: close 1001 (going away)" {
					return
				}
				conn.WriteMessage(t, []byte(fmt.Sprint(err)))
			}

		}
	}()
	for range ticker.C {
		//log.Println("websocket ok")
		//log.Println(gin.H{
		//	"ProcessInfo":GetProcess_Info(),
		//})
		info, err := GetProcess_Info()
		if err == nil {
			err := conn.WriteJSON(gin.H{
				"ProcessInfo": info,
			})
			if err != nil {
				//log.Println("websocket disconnect")
				conn.Close()
				break
			}
		}
	}
}

func ProcessHandler_get(c *gin.Context) {
	c.HTML(http.StatusOK, "processes.html", gin.H{})
}

func PathHandler_get(c *gin.Context) {
	path := c.DefaultQuery("path", "/")
	var dirs []DirItem
	var files []FileItem
	var writable bool
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			dirs = make([]DirItem, 0)
			files = make([]FileItem, 0)
			writable = false
		}
	} else {
		if !stat.IsDir() {
			dirs = make([]DirItem, 0)
			files = make([]FileItem, 0)
			writable = false
		} else {
			allfiles, err := ioutil.ReadDir(path)
			if err == nil {
				dirs = GetDirs(path, allfiles)
				files = GetFiles(path, allfiles)
				writable = unix.Access(path, unix.W_OK) == nil
			} else {
				dirs = make([]DirItem, 0)
				files = make([]FileItem, 0)
				writable = false
			}
		}
	}
	c.HTML(http.StatusOK, "path.html", gin.H{"header": path, "writable": writable, "dirs": dirs, "files": files})
}

func DownloadHandler_get(c *gin.Context) {
	path := c.DefaultQuery("name", "")
	if len(path) == 0 {
		c.String(http.StatusBadRequest, "invalid url")
	} else {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", "attachment; filename=files.zip")
		c.Header("Content-Type", "application/octet-stream")
		defer func() {
			os.Remove(path)
		}()
		c.File(path)
	}
}

func DownloadHandler_post(c *gin.Context) {
	var filelist FileList
	err := c.BindJSON(&filelist)
	if err == nil {
		for i, file := range filelist.Files {
			path, err := url.QueryUnescape(file)
			if err != nil {
				logErr(err)
				c.String(http.StatusBadRequest, fmt.Sprint(err))
				return
			}
			filelist.Files[i] = path
		}
		if len(filelist.Files) == 0 {
			logErr(err)
			c.String(http.StatusBadRequest, fmt.Sprint(err))
			return
		}
		openfiles := make([]*os.File, 0)
		for _, filename := range filelist.Files {
			f, err := os.Open(filename)
			defer f.Close()
			if err != nil {
				logErr(err)
				c.String(http.StatusBadRequest, fmt.Sprint(err))
				return
			}
			openfiles = append(openfiles, f)
		}
		ex, err := os.Executable()
		if err != nil {
			logErr(err)
			c.String(http.StatusBadRequest, fmt.Sprint(err))
			return
		}
		expath := filepath.Dir(ex)
		zippath := filepath.Join(expath, uuid.Must(uuid.NewV4()).String()+".zip")
		err = Compress(openfiles, zippath)
		if err != nil {
			logErr(err)
			os.Remove(zippath)
			c.String(http.StatusBadRequest, fmt.Sprint(err))
			return
		}
		c.String(http.StatusOK, "ok "+zippath)
	} else {
		logErr(err)
		c.String(http.StatusBadRequest, fmt.Sprint(err))
	}
}

func UploadHandler_post(c *gin.Context) {
	path := c.DefaultPostForm("path", "")
	if len(path) == 0 {
		c.String(http.StatusBadRequest, "invalid path")
	} else {
		file, header, err := c.Request.FormFile("files")
		checkErr(err)
		filename := header.Filename
		out, err := os.Create(filepath.Join(path, filename))
		checkErr(err)
		defer out.Close()
		_, err = io.Copy(out, file)
		checkErr(err)
		c.JSON(http.StatusOK, gin.H{"name": filename})
	}
}

func DeleteHandler_post(c *gin.Context) {
	var filelist FileList
	err := c.BindJSON(&filelist)
	if err == nil {
		for i, file := range filelist.Files {
			path, err := url.QueryUnescape(file)
			if err != nil {
				logErr(err)
				c.String(http.StatusBadRequest, fmt.Sprint(err))
				return
			}
			filelist.Files[i] = path
		}
		if len(filelist.Files) == 0 {
			logErr(err)
			c.String(http.StatusBadRequest, fmt.Sprint(err))
			return
		}
		for _, file := range filelist.Files {
			err = os.RemoveAll(file)
			if err != nil {
				logErr(err)
				c.String(http.StatusBadRequest, fmt.Sprint(err))
				return
			}
		}
		c.JSON(http.StatusOK, "ok")
	} else {
		logErr(err)
		c.String(http.StatusBadRequest, fmt.Sprint(err))
	}
}

func AboutHandler_get(c *gin.Context) {
	c.HTML(http.StatusOK, "about.html", gin.H{})
}

func AuthorsHandler_get(c *gin.Context) {
	c.HTML(http.StatusOK, "authors.html", gin.H{})
}
