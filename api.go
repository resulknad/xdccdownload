
package main
//import "time"
//import "fmt"
//import "os"
import "github.com/gin-gonic/gin"
//import "container/list"
import "strconv"
import "strings"


type DownloadManagerRestAPI struct {
    DownloadManager
}

func (dm *DownloadManagerRestAPI) ListAll(c *gin.Context) {
    c.JSON(200, dm.AllDownloads())
}

func (dm *DownloadManagerRestAPI) GetParamOrThrow(c *gin.Context) int {
    i, err := strconv.Atoi(c.Params.ByName("id"))
    if err != nil {
        c.String(500, "Error")
        return -1
    }
    return i
}

func (dm *DownloadManagerRestAPI) Create(c *gin.Context) {
    var d Download
    c.Bind(&d)
    found, pack:= dm.Indx.GetPackage(d.Packid)
    if found {
        d.Pack = pack
        dm.CreateDownload(d)
    }
    dm.ListAll(c)
}

func (dm *DownloadManagerRestAPI) Update(c *gin.Context) {
    //i := dm.GetParamOrThrow(c)
	i := c.Params.ByName("id")
    found, _ := dm.GetDownload(i)
    if found>-1 {
    }
}

func (dm *DownloadManagerRestAPI) Delete(c *gin.Context) {
    //i := dm.GetParamOrThrow(c)
	i := c.Params.ByName("id")
    dm.DeleteOne(i)
    dm.ListAll(c)
}

type IndexerEndpoints struct{
    *Indexer
}

func (ie *IndexerEndpoints) pkgQuery(c *gin.Context) {
    c.JSON(200,ie.Search("%" + strings.Replace(c.Param("query"), " ", "%", -1) + "%"))
}
