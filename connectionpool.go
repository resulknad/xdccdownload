package main
import "sync"
import "log"

type ConnectionBackoff struct {
	Server string
	Backoff int
}

type ConnectionPool struct {
    connections []*IRC
	backoff []ConnectionBackoff
    sync.Mutex
}

func (c *ConnectionPool) GetBackoffAndIncrement(Server string) int {
	for _,b := range(c.backoff) {
		if b.Server == Server {
			b.Backoff++
			return b.Backoff
		}
	}
	c.backoff = append(c.backoff, ConnectionBackoff{Server, 1})
	return 1
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

func (c *ConnectionPool) Quit(server string) {
	log.Print("quitting " + server)
    c.Lock()
    defer c.Unlock()
    for indx, irc := range(c.connections) {
        if irc.Server == server {
			log.Print("quitting connection")
            irc.Quit()
			log.Print(c.connections)
			c.connections = append(c.connections[:indx], c.connections[indx+1:]...)
			log.Print(c.connections)
			break
        }
    }
}


