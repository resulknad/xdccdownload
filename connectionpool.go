package main
import "sync"

type ConnectionPool struct {
    connections []*IRC
    sync.Mutex
}

func (c *ConnectionPool) GetConnection(Server string) *IRC {
    c.Lock()
    defer c.Unlock()
    for _, irc := range(c.connections) {
        if irc.Server == Server {
            return irc
        }
    }
    irc := IRC{Server: Server}
    if irc.Connect() {
        c.connections = append(c.connections, &irc)
        return &irc
    } else {
        return nil
    }
}

