package main

import "github.com/mholt/archiver"
import "path"
import "path/filepath"
import "github.com/elgs/gostrgen"
import "github.com/jagadeesh-kotra/gorar"
import "os"
import "fmt"

type Unpack struct {
    TempDir string
    TargetDir string
    InitialPath string
}

func (u *Unpack) Unpack(archivePath string) {
    randomFolder, _ := gostrgen.RandGen(15, gostrgen.Lower | gostrgen.Upper, "", "")

    randomPath := path.Join(u.TempDir, randomFolder)
    fmt.Println(randomPath)

    if path.Ext(archivePath) == ".rar" {
        gorar.RarExtractor(archivePath, randomPath)
    } else if path.Ext(archivePath)[0:2] != ".r" {
        err := archiver.MatchingFormat(archivePath).Open(archivePath, randomPath)
        if (err != nil) {
            panic(err)
        }
    }
    filepath.Walk(randomPath, u.Process)
    os.RemoveAll(randomPath)
}

func (u *Unpack) DesiredFile(filePath string) bool {
    switch path.Ext(filePath) {
        case ".mp3", ".mp4", ".srt", ".mkv", ".avi":
            return true
        }
        return false
}

func (u *Unpack) Do() {
    u.Process(u.InitialPath, nil, nil)
    os.Remove(u.InitialPath)
}

func (u *Unpack) Process(filePath string, info os.FileInfo, err error) error {
    if (archiver.MatchingFormat(filePath) != nil) {
        u.Unpack(filePath)
    } else if (u.DesiredFile(filePath)) {
        os.MkdirAll(u.TargetDir, 0777)
        os.Rename(filePath, path.Join(u.TargetDir, path.Base(filePath)))
    }
    return nil
}
