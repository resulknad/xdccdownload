package main

import "github.com/mholt/archiver"
import "io"
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


    if path.Ext(archivePath) == ".rar" {
		fmt.Println(randomPath)
        gorar.RarExtractor(archivePath, randomPath)
    } else if path.Ext(archivePath)[0:2] != ".r" {
        err := archiver.Unarchive(archivePath, randomPath)
        if (err != nil) {
            panic(err)
        }
    }
    filepath.Walk(randomPath, u.Process)
	os.RemoveAll(randomPath)
}

func (u *Unpack) DesiredFile(filePath string) bool {
  fmt.Println(filePath)
    switch path.Ext(filePath) {
        case ".mp3", ".mp4", ".srt", ".mkv", ".avi":
            return true
        }
        return false
}

func (u *Unpack) Do() (suc bool) {
	suc = false
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in f", r)
        }
    }()
    u.Process(u.InitialPath, nil, nil)
    os.Remove(u.InitialPath)
    suc = true
	return
}

func copy_file(src, dst string) (int64, error) {
        sourceFileStat, err := os.Stat(src)
        if err != nil {
                return 0, err
        }

        if !sourceFileStat.Mode().IsRegular() {
                return 0, fmt.Errorf("%s is not a regular file", src)
        }

        source, err := os.Open(src)
        if err != nil {
                return 0, err
        }
        defer source.Close()

        destination, err := os.Create(dst)
        if err != nil {
                return 0, err
        }
        defer destination.Close()
        nBytes, err := io.Copy(destination, source)
        return nBytes, err
}

func (u *Unpack) Process(filePath string, info os.FileInfo, err error) error {
	_, err2 := archiver.ByExtension(filePath)
    if (err2 == nil) {
        u.Unpack(filePath)
    } else if (u.DesiredFile(filePath)) {
        os.MkdirAll(u.TargetDir, 0777)

		//os.Rename(filePath, path.Join(u.TargetDir, path.Base(filePath)))
		copy_file(filePath, path.Join(u.TargetDir, path.Base(filePath)))
		os.Remove(filePath)
    } else {
	  fmt.Println("3rd case")
	}
    return nil
}
