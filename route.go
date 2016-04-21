package main

import (
	"bytes"
	"fmt"
	"strings"
)

type Config struct {
	Hostname string
}

func AddRoute(config Config) error {
	err := run("route", "-n", "add", "172.17.0.0/16", config.Hostname)
	if err != nil {
		return err
	}

	nameBytes, err := runOutput("route", "-n", "get", config.Hostname)
	if err != nil {
		return err
	}

	nameStartMarker := []byte("interface: ")
	nameEndMarker := []byte("\n      flags")

	nameStart := bytes.Index(nameBytes, nameStartMarker)
	nameEnd := bytes.Index(nameBytes, nameEndMarker)

	if nameStart == -1 || nameEnd == -1 {
		return fmt.Errorf("Unable to add route")
	}

	name := string(nameBytes[nameStart+len(nameStartMarker) : nameEnd])

	memberBytes, err := runOutput("ifconfig", name)
	if err != nil {
		return err
	}

	memberStartMarker := []byte("member: ")
	memberEndMarker := []byte(" flags=")

	memberStart := bytes.Index(memberBytes, memberStartMarker)
	if memberStart == -1 {
		return fmt.Errorf("Unable to add route")
	}

	memberEnd := bytes.Index(memberBytes[memberStart:], memberEndMarker) + memberStart
	if memberEnd == -1 {
		return fmt.Errorf("Unable to add route")
	}

	members := strings.Split(string(memberBytes[memberStart+len(memberStartMarker):memberEnd]), " ")
	for _, member := range members {
		err := run("ifconfig", name, "-hostfilter", member)
		if err != nil {
			return err
		}
	}

	return nil
}
