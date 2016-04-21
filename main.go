package main

import (
	"log"
	"os"
	"os/user"
	"time"

	"github.com/ayufan/golang-kardianos-service"

	"golang.org/x/net/context"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	app := kingpin.New("docker-route", "Docker for Mac route helper")

	var (
		runCmd       = app.Command("run", "run docker-route")
		startCmd     = app.Command("start", "start docker-route")
		stopCmd      = app.Command("stop", "stop docker-route")
		restartCmd   = app.Command("restart", "restart docker-route")
		installCmd   = app.Command("install", "install docker-route")
		uninstallCmd = app.Command("uninstall", "uninstall docker-route")
		statusCmd    = app.Command("status", "status docker-route")
		asUser       string
	)

	runCmd.Arg("user", "user to run the helper for").StringVar(&asUser)

	if os.Getenv("USER") != "root" {
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		if usr.Username != "root" {
			log.Fatal("must be run as root")
		}
	}

	var (
		svc service.Service
		err error
		cnf = &service.Config{
			Name:        "docker-route",
			DisplayName: "Docker Route Helper",
			Description: "Manage docker for mac routes",
			UserName:    "root",
			Arguments:   []string{"run"},
			Option: service.KeyValue{
				"KeepAlive": true,
				"RunAtLoad": true,
			},
		}
	)

	helper := newHelper()
	svc, err = service.New(helper, cnf)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case runCmd.FullCommand():
		helper.User = asUser
		err = svc.Run()
	case startCmd.FullCommand():
		err = svc.Start()
	case stopCmd.FullCommand():
		err = svc.Stop()
	case restartCmd.FullCommand():
		err = svc.Restart()
	case installCmd.FullCommand():
		if u := os.Getenv("SUDO_USER"); u == "" {
			log.Fatal("unable to detect SUDO user")
		} else {
			cnf.Arguments = append(cnf.Arguments, u)
		}
		err = svc.Install()
	case uninstallCmd.FullCommand():
		err = svc.Uninstall()
	case statusCmd.FullCommand():
		err = svc.Status()
	}

	if err != nil {
		log.Fatal(err)
	}
}

type helper struct {
	User   string
	ctx    context.Context
	cancel func()
}

func newHelper() *helper {
	ctx, cancel := context.WithCancel(context.Background())
	return &helper{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (h *helper) Start(s service.Service) error {
	errs := make(chan error, 5)
	logger, err := s.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()

	logger.Errorf("Running for %q", h.User)

	go func() {
		defer h.cancel()

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		err := setup(h.User)
		if err != nil {
			logger.Error(err)
		}

		for {
			select {

			case <-ticker.C:
				err := setup(h.User)
				if err != nil {
					logger.Error(err)
				}

			case <-h.ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (h *helper) Stop(s service.Service) error {
	h.cancel()
	return nil
}
