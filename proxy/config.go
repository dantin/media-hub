package proxy

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type mirrorItem struct {
	ipAddr string
	port   int
}

// MirrorList represents comma separated mirror items.
type MirrorList []mirrorItem

func (l *MirrorList) String() string {
	return fmt.Sprint(*l)
}

// Set confirm flag Set interface.
func (l *MirrorList) Set(value string) error {
	for _, m := range strings.Split(value, ",") {
		tokens := strings.Split(m, ":")
		if len(tokens) != 2 {
			log.Printf("bad format of mirror item %s", m)
		}
		port, err := strconv.Atoi(tokens[1])
		if err != nil {
			log.Printf("bad port number of mirror item %s, caused by: %s", m, err)
			continue
		}

		*l = append(*l, mirrorItem{
			ipAddr: tokens[0],
			port:   port,
		})
	}
	return nil
}
