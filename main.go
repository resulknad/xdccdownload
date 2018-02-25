package main
import "time"
import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/cors"

func main() {
    connPool := ConnectionPool{}
    c := Config{}
    c.LoadConfig()

    var indx *Indexer
    for indx == nil {
	    indx = CreateIndexer(&c, &connPool)
	    time.Sleep(1*time.Second)
    }

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
