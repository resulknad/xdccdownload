package main
import "time"
import "fmt"
import "os"
import "path"
import "github.com/gin-gonic/gin"
import "github.com/gin-contrib/cors"

func main() {
    c := Config{}
    c.LoadConfig()
    indx := CreateIndexer(&c)
    ie := IndexerEndpoints{indx}
    dm := DownloadManagerRestAPI{*CreateDownloadManager(indx, &c)}
    router := gin.Default()
    router.Use(cors.Default())

    // determine ui path
    execPath, err := os.Executable()
    if (err != nil) { panic(err) }
    uiPath := path.Join(execPath, "ui")
    fmt.Println(uiPath)
    StaticServe(router)
    //router.Use(static.Serve("/", static.LocalFile(uiPath, false)))
    router.GET("/config/", func (g *gin.Context) { g.JSON(200, c) })
    router.GET("/packages/:query", ie.pkgQuery)
    router.GET("/download/", dm.ListAll)
    router.DELETE("/download/:id", dm.Delete)
    router.POST("/download/", dm.Create)
    router.PUT("/download/:id", dm.Update)

    router.Run(":8080")
    for {
        time.Sleep(10*time.Second)
    }
}
