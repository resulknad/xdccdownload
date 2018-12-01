package main
import "time"
import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/cors"
import "log"
import "path"
import "os"
import "flag"

func main() {

	var printLog = flag.Int("printLog", 0, "print log to stdout")
	flag.Parse()

    c := Config{}
    c.LoadConfig()

	connPool := ConnectionPool{c: &c}
	
	logFile := path.Join(c.LogDir, time.Now().Format("20060102"))
	if _, err := os.Stat(logFile); err == nil {

	}

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}   
	defer f.Close()

	if *printLog == 0 {
		log.SetOutput(f)
	}

    var indx *Indexer
    //for indx == nil {
	    indx = CreateIndexer(&c, &connPool)
	//    time.Sleep(1*time.Second)
    //}

	if len(os.Args) > 1 && os.Args[1] == "update" {
	  imdb := indx.imdb
		imdb.UpdateData()
		indx.EnrichAll()
	}

	indx.InitWatchDog()

    // configure rest api handlers
    ie := IndexerEndpoints{indx}
	dlm := CreateDownloadManager(indx, &c, &connPool)
    dm := DownloadManagerRestAPI{dlm}


// Creates a router without any middleware by default
	router := gin.New()
	router.Use(gin.Recovery())

    config := cors.DefaultConfig()
    config.AllowAllOrigins = true
    config.AddAllowMethods("DELETE")
    router.Use(cors.New(config))


    // files embedded with bin data
    StaticServe(router)

    // specify endpoints
    router.GET("/config/", func (g *gin.Context) { g.JSON(200, c) })
    router.GET("/packages/:query", ie.pkgQuery)
    router.GET("/resetdownloaded/", ie.ResetDlDb)
    router.GET("/download/", dm.ListAll)
    router.DELETE("/download/:id", dm.Delete)
    router.POST("/download/", dm.Create)
    router.PUT("/download/:id", dm.Update)

	
	tm := CreateTaskmgr(indx, dlm, &c)
	go tm.StartAllTasks()
    tm_api := TaskmgrEndpoints{tm}
	router.GET("/tasks/", tm_api.All)
	router.POST("/tasks/", tm_api.Create)
	router.GET("/tasks/:id", tm_api.Get)
	router.DELETE("/tasks/:id", tm_api.Delete)
	router.PUT("/tasks/:id", tm_api.Update)

    router.Run(":" + string(c.Port))
    for {
        time.Sleep(10*time.Second)
    }
}
