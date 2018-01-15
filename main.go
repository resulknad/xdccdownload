package main
import "time"
import "fmt"
import "os"
import "github.com/urfave/cli"

func main() {
    app := cli.NewApp()
      app.Commands = []cli.Command{
    {
      Name:    "complete",
      Aliases: []string{"c"},
      Usage:   "complete a task on the list",
      Action:  func(c *cli.Context) error {
        return nil
      },
    },
    {
      Name:    "add",
      Aliases: []string{"a"},
      Usage:   "add a task to the list",
      Action:  func(c *cli.Context) error {
        return nil
      },
    },
  }
  app.Run(os.Args)
    CreateIndexer()
    for {
        time.Sleep(10*time.Second)
    }
    time.Sleep(40*time.Second)
    fmt.Println("Usage: [SERVER]:[PORT] [CHANNEL] [BOT] [PACKAGE]")
    fmt.Println(os.Args)
    i := IRC{Server: os.Args[1], Nick: "asdfasaf213d"}
    i.Connect()
    fmt.Println(os.Args)
    x := XDCC{Bot: os.Args[3], Channel: os.Args[2], Package: os.Args[4], IRCConn: &i}
    x.Download()
    fmt.Print(x)
}
