package main

import (
	"os"
	"strings"

	"github.com/bingoohuang/gosnmpd"
	"github.com/bingoohuang/gosnmpd/mibImps"
	"github.com/sirupsen/logrus"
	"github.com/slayercat/gosnmp"
	"github.com/urfave/cli/v2"
)

func makeApp() *cli.App {
	return &cli.App{
		Name:        "gosnmpd",
		Description: "an example server of gosnmp",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "run-server",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "logLevel", Value: "info"},
					&cli.StringFlag{Name: "community", Value: "public"},
					&cli.StringFlag{Name: "bindTo", Value: "127.0.0.1:1161"},
					&cli.StringFlag{Name: "v3Username", Value: "testuser"},
					&cli.StringFlag{Name: "v3AuthenticationPassphrase", Value: "testauth"},
					&cli.StringFlag{Name: "v3PrivacyPassphrase", Value: ""},
				},
				Action: runServer,
			},
		},
	}
}

func main() {
	app := makeApp()
	app.Run(os.Args)
}

func runServer(c *cli.Context) error {
	logger := gosnmpd.NewDefaultLogger()
	switch strings.ToLower(c.String("logLevel")) {
	case "fatal":
		logger.(*gosnmpd.DefaultLogger).Level = logrus.FatalLevel
	case "error":
		logger.(*gosnmpd.DefaultLogger).Level = logrus.ErrorLevel
	case "info":
		logger.(*gosnmpd.DefaultLogger).Level = logrus.InfoLevel
	case "debug":
		logger.(*gosnmpd.DefaultLogger).Level = logrus.DebugLevel
	case "trace":
		logger.(*gosnmpd.DefaultLogger).Level = logrus.TraceLevel
	}
	mibImps.SetupLogger(logger)

	userName := c.String("v3Username")
	authenticationPassphrase := c.String("v3AuthenticationPassphrase")
	privacyPassphrase := c.String("v3PrivacyPassphrase")
	privacyProtocol := gosnmp.DES
	if privacyPassphrase == "" {
		privacyProtocol = gosnmp.NoPriv
	}

	users := gosnmp.UsmSecurityParameters{
		UserName:                 userName,
		AuthenticationProtocol:   gosnmp.MD5,
		PrivacyProtocol:          privacyProtocol,
		AuthenticationPassphrase: authenticationPassphrase,
		PrivacyPassphrase:        privacyPassphrase,
	}

	master := gosnmpd.MasterAgent{
		Logger: logger,
		SecurityConfig: gosnmpd.SecurityConfig{
			AuthoritativeEngineBoots: 1,
			Users:                    []gosnmp.UsmSecurityParameters{users},
		},
		SubAgents: []*gosnmpd.SubAgent{
			{
				CommunityIDs: []string{c.String("community")},
				OIDs:         mibImps.All(),
			},
		},
	}
	logger.Infof("V3 Users:")
	for _, val := range master.SecurityConfig.Users {
		logger.Infof(
			"\tUserName:%v\n\t -- AuthenticationProtocol:%v\n\t -- PrivacyProtocol:%v\n\t -- AuthenticationPassphrase:%v\n\t -- PrivacyPassphrase:%v",
			val.UserName,
			val.AuthenticationProtocol,
			val.PrivacyProtocol,
			val.AuthenticationPassphrase,
			val.PrivacyPassphrase,
		)
	}
	server := gosnmpd.NewSNMPServer(master)
	err := server.ListenUDP("udp", c.String("bindTo"))
	if err != nil {
		logger.Errorf("Error in listen: %+v", err)
	}
	server.ServeForever()
	return nil
}
