package server

import (
	"context"
	"im/container"
	"im/core"
	"im/iface"
	"im/logger"
	"im/naming"
	"im/naming/consul"
	"im/services/server/conf"
	"im/services/server/handler"
	"im/services/server/serv"
	"im/storage"
	"im/tcp"

	"github.com/klintcheng/kim/wire"
	"github.com/spf13/cobra"
)

type ServerStartOptions struct {
	config      string
	serviceName string
}

func NewServerStartCmd(ctx context.Context, version string) *cobra.Command {
	opts := &ServerStartOptions{}
	cmd := &cobra.Command{
		Use:   "server",
		Short: "启动chat服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServerStart(ctx, opts, version)
		},
	}
	cmd.PersistentFlags().StringVarP(&opts.config, "config", "c", "./server/conf.yaml", "Config file")
	cmd.PersistentFlags().StringVarP(&opts.serviceName, "serviceName", "s", "chat", "defined a service name,option is login or chat")
	return cmd
}

func RunServerStart(ctx context.Context, opts *ServerStartOptions, version string) error {
	config, err := conf.Init(opts.config)
	if err != nil {
		return err
	}
	logger.Init(logger.Settings{
		Level: "trace",
	})
	r := core.NewRouter()
	//handler
	loginHandler := handler.NewLoginHandler()
	r.Handle(wire.CommandLoginSignIn, loginHandler.DoSysLogin)
	r.Handle(wire.CommandLoginSignOut, loginHandler.DoSysLogout)

	rdb, err := conf.InitRedis(config.RedisAddrs, "")
	if err != nil {
		return err
	}
	cache := storage.NewRedisStoreage(rdb)
	servhandler := serv.NewServHandler(r, cache)

	service := &naming.DefaultService{
		Id:       config.ServiceID,
		Name:     opts.serviceName,
		Address:  config.PublicAddress,
		Port:     config.PublicPort,
		Protocol: string(wire.ProtocolTCP),
		Tags:     config.Tags,
	}

	srv := tcp.NewServer(config.Listen, service)
	srv.SetReadWait(iface.DefaultReadWait)
	srv.SetAcceptor(servhandler)
	srv.SetMessageListener(servhandler)
	srv.SetStateListener(servhandler)

	if err := container.Init(srv); err != nil {
		return err
	}

	ns, err := consul.NewNaming(config.ConsulURL)
	if err != nil {
		return err
	}
	container.SetServiceNaming(ns)
	return container.Start()
}
