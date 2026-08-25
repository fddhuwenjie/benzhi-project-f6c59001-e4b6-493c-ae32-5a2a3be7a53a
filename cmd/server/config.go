package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultAddress = "127.0.0.1:19081"

type config struct {
	address  string
	dataDir  string
	selftest bool
}

func parseConfig(args []string, lookupEnv func(string) string) (config, error) {
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	var address string
	var dataDir string
	var selftest bool
	flags.StringVar(&address, "addr", "", "HTTP 监听地址")
	flags.StringVar(&dataDir, "data", "./data", "持久化数据目录")
	flags.BoolVar(&selftest, "selftest", false, "执行完整 HTTP 流程自检后退出")
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("未知位置参数: %s", strings.Join(flags.Args(), " "))
	}
	if address == "" {
		portValue := strings.TrimSpace(lookupEnv("PORT"))
		if portValue == "" {
			address = defaultAddress
		} else {
			port, err := strconv.Atoi(portValue)
			if err != nil || port < 1 || port > 65535 {
				return config{}, errors.New("PORT 必须是 1 到 65535 之间的端口号")
			}
			address = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		}
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil || strings.TrimSpace(host) == "" || port == "" {
		return config{}, fmt.Errorf("-addr 必须是 host:port 格式: %w", err)
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return config{}, errors.New("-addr 端口必须是 1 到 65535 之间的数字")
	}
	if strings.TrimSpace(dataDir) == "" {
		return config{}, errors.New("-data 不能为空")
	}
	return config{address: address, dataDir: dataDir, selftest: selftest}, nil
}

func environment(name string) string { return os.Getenv(name) }
