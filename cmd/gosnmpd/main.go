package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/bingoohuang/gg/pkg/v"
	"github.com/bingoohuang/godaemon"
	"github.com/bingoohuang/gosnmpd"
	"github.com/bingoohuang/gosnmpd/mibImps"
	"github.com/sirupsen/logrus"
	"github.com/slayercat/gosnmp"
	"github.com/spf13/pflag"
)

type App struct {
	Version      bool   `flag:"version,v" value:"false" usage:"print version and exit"`
	Deamon       bool   `flag:"deamon,d" value:"false" usage:"run as daemon"`
	LogLevel     string `flag:"logLevel,l" value:"info"`
	Community    string `flag:"community,c" value:"public"`
	BindTo       string `flag:"bindTo" value:":1161"`
	User         string `flag:"userName,u" value:"testuser"`
	AuthProtocol string `flag:"authProtocol,a" value:"md5" usage:"SnmpV3AuthProtocol, md5/sha/none"`
	AuthPass     string `flag:"authPass,A" value:"testauth"`
	PriProtocol  string `flag:"priProtocol,x" value:"none" usage:"SnmpV3PrivProtocol, des/aes/none"`
	PriPass      string `flag:"priPass,X" value:""`
}

func main() {
	var app App
	declareFlags(&app)
	pflag.Parse()

	if app.Version {
		fmt.Println(v.Version())
		os.Exit(0)
	}

	godaemon.Deamonize(app.Deamon)

	app.runServer()
}

func (a *App) runServer() error {
	logger := gosnmpd.NewDefaultLogger().(*gosnmpd.DefaultLogger)
	logger.Level = a.parseLevel()
	mibImps.SetupLogger(logger)

	var priProtocol gosnmp.SnmpV3PrivProtocol
	switch strings.ToLower(a.PriProtocol) {
	case "des":
		priProtocol = gosnmp.DES
	case "aes":
		priProtocol = gosnmp.AES
	case "aes192":
		priProtocol = gosnmp.AES192
	case "aes256":
		priProtocol = gosnmp.AES256
	case "aes192c":
		priProtocol = gosnmp.AES192C
	case "aes256c":
		priProtocol = gosnmp.AES256C
	case "none":
		priProtocol = gosnmp.NoPriv
	}

	if a.PriPass == "" {
		priProtocol = gosnmp.NoPriv
	} else if priProtocol <= gosnmp.NoPriv {
		priProtocol = gosnmp.DES
	}

	var authProtocol gosnmp.SnmpV3AuthProtocol
	switch strings.ToLower(a.AuthProtocol) {
	case "md5":
		authProtocol = gosnmp.MD5
	case "sha":
		authProtocol = gosnmp.SHA
	case "none", "":
		authProtocol = gosnmp.NoAuth
	}
	if a.AuthPass == "" {
		authProtocol = gosnmp.NoAuth
	} else if authProtocol <= gosnmp.NoAuth {
		authProtocol = gosnmp.MD5
	}

	users := gosnmp.UsmSecurityParameters{
		UserName:                 a.User,
		AuthenticationProtocol:   authProtocol,
		AuthenticationPassphrase: a.AuthPass,
		PrivacyProtocol:          priProtocol,
		PrivacyPassphrase:        a.PriPass,
	}

	master := gosnmpd.MasterAgent{
		Logger: logger,
		SecurityConfig: gosnmpd.SecurityConfig{
			AuthoritativeEngineBoots: 1,
			Users:                    []gosnmp.UsmSecurityParameters{users},
		},
		SubAgents: []*gosnmpd.SubAgent{
			{
				CommunityIDs: []string{a.Community},
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
	err := server.ListenUDP("udp", a.BindTo)
	if err != nil {
		logger.Errorf("Error in listen: %+v", err)
	}
	server.ServeForever()
	return nil
}

func (a *App) parseLevel() logrus.Level {
	switch strings.ToLower(a.LogLevel) {
	case "fatal":
		return logrus.FatalLevel
	case "error":
		return logrus.ErrorLevel
	case "info":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	case "trace":
		return logrus.TraceLevel
	}

	return logrus.InfoLevel
}

func declareFlags(app *App) {
	rv := reflect.ValueOf(app).Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if !ft.IsExported() {
			continue
		}

		flag := ft.Tag.Get("flag")
		if flag == "-" {
			continue
		}

		short := ""
		if j := strings.Index(flag, ","); j >= 0 {
			short = flag[j+1:]
			flag = flag[:j]
		}

		if flag == "" {
			flag = strings.ToLower(ft.Name[:1]) + ft.Name[1:]
		}

		fv := rv.Field(i).Addr().Interface()
		usage := ft.Tag.Get("usage")
		value := ft.Tag.Get("value")

		switch ft.Type.Kind() {
		case reflect.Bool:
			pflag.BoolVarP(fv.(*bool), flag, short, "true" == value, usage)
		case reflect.String:
			pflag.StringVarP(fv.(*string), flag, short, value, usage)
		}
	}
}
