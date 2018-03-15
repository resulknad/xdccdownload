package main
import "time"
import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/cors"
import "log"
import "path"
import "os"

func main() {

    connPool := ConnectionPool{}
    c := Config{}
    c.LoadConfig()
	
	logFile := path.Join(c.LogDir, time.Now().Format("20060102"))
	if _, err := os.Stat(logFile); err == nil {

	}

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}   
	defer f.Close()
	log.SetOutput(f)

    var indx *Indexer
    for indx == nil {
	    indx = CreateIndexer(&c, &connPool)
	    time.Sleep(1*time.Second)
    }

	indx.InitWatchDog()

    // configure rest api handlers
    ie := IndexerEndpoints{indx}
    dm := DownloadManagerRestAPI{*CreateDownloadManager(indx, &c, &connPool)}
    router := gin.Default()
    config := cors.DefaultConfig()
    config.AllowAllOrigins = true
    config.AddAllowMethods("DELETE")
    router.Use(cors.New(config))


    // files embedded with bin data
    StaticServe(router)

    // specify endpoints
    router.GET("/config/", func (g *gin.Context) { g.JSON(200, c) })
    router.GET("/packages/:query", ie.pkgQuery)
    router.GET("/download/", dm.ListAll)
    router.DELETE("/download/:id", dm.Delete)
    router.POST("/download/", dm.Create)
    router.PUT("/download/:id", dm.Update)

    router.Run(":" + string(c.Port))
    for {
        time.Sleep(10*time.Second)
    }
}
