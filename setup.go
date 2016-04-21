package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
)

func setup(user string) error {
	ip, err := net.ResolveIPAddr("ip4", "docker.local")
	if err != nil {
		return fmt.Errorf("lookup %q failed %s", "docker.local", err)
	}

	out, err := runOutput("sudo", "-u", user, "-i", "-i", "docker", "ps", "-f", "name=etcd", "-q")
	if err != nil {
		return fmt.Errorf("lookup of %q container failed %s", "etcd", err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		log.Printf("starting etcd")
		err = run("sudo", "-u", user, "-i", "docker", "run",
			"-d",
			"--name", "etcd",
			"--restart", "always",
			"mrhenry/etcd:b1")
		if err != nil {
			return err
		}
	}

	out, err = runOutput("sudo", "-u", user, "-i", "-i", "docker", "ps", "-f", "name=skydns", "-q")
	if err != nil {
		return fmt.Errorf("lookup of %q container failed %s", "skydns", err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		log.Printf("starting skydns")
		err = run("sudo", "-u", user, "-i", "docker", "run",
			"-d",
			"--name", "skydns",
			"--link", "etcd",
			"-p", "172.17.0.1:53:53/udp",
			"--restart", "always",
			"mrhenry/skydns:b1")
		if err != nil {
			return err
		}
	}

	out, err = runOutput("sudo", "-u", user, "-i", "-i", "docker", "ps", "-f", "name=switch", "-q")
	if err != nil {
		return fmt.Errorf("lookup of %q container failed %s", "switch", err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		log.Printf("starting switch")
		err = run("sudo", "-u", user, "-i", "docker", "run",
			"-d",
			"--name", "switch",
			"--link", "etcd",
			"-v", "/var/run/docker.sock:/var/run/docker.sock",
			"--restart", "always",
			"mrhenry/switch:b1")
		if err != nil {
			return err
		}
	}

	// cnfData, err := runOutput("sudo", "-u", user, "-i", "pinata", "get", "daemon")
	// if err != nil {
	// 	return err
	// }
	//
	// var cnf map[string]interface{}
	//
	// err = json.Unmarshal(cnfData, &cnf)
	// if err != nil {
	// 	return err
	// }
	//
	// var (
	// 	bip, _  = cnf["bip"].(string)
	// 	dns, _  = cnf["dns"].([]interface{})
	// 	changed bool
	// )
	//
	// if bip != "172.17.0.1/24" {
	// 	changed = true
	// 	cnf["bip"] = "172.17.0.1/24"
	// }
	// if len(dns) != 1 {
	// 	changed = true
	// 	cnf["dns"] = []string{"172.17.0.1"}
	// }
	//
	// if changed {
	// 	log.Printf("configuring docker")
	//
	// 	data, err := json.Marshal(cnf)
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	fmt.Printf("config: %s\n", data)
	//
	// 	err = runInput(data, "sudo", "-u", user, "-i", "pinata", "set", "daemon", "-")
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	err = run("sudo", "-u", user, "-i", "pinata", "restart")
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	log.Printf("routing 172.17.x.x to %s", ip)
	err = AddRoute(Config{
		Hostname: ip.String(),
	})
	if err != nil {
		return err
	}

	err = run("mkdir", "-p", "/etc/resolver")
	if err != nil {
		return err
	}

	err = runInput([]byte("nameserver 172.17.0.1\n"), "tee", "/etc/resolver/switch")
	if err != nil {
		return err
	}

	err = run("dscacheutil", "-flushcache")
	if err != nil {
		return err
	}

	err = run("killall", "-HUP", "mDNSResponder")
	if err != nil {
		return err
	}

	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func runInput(data []byte, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
