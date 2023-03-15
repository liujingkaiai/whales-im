package service

import (
	"context"
	"fmt"
	"hash/crc32"
	"im/logger"
	"im/naming"
	"im/naming/consul"
	"im/services/service/conf"
	"im/services/service/database"
	"im/services/service/handler"

	"github.com/kataras/iris/v12"
	"github.com/klintcheng/kim/wire"
)

type ServerStartOptions struct {
	config string
}

func RunServerStart(ctx context.Context, opts *ServerStartOptions, version string) error {
	config, err := conf.Init(opts.config)
	if err != nil {
		return err
	}
	logger.Init(logger.Settings{
		Level:    config.LogLevel,
		Filename: "./data/royal.log",
	})

	db, err := database.InitMysqlDb(config.BaseDb)
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&database.Group{}, &database.GroupMember{})
	if err != nil {
		return err
	}

	messageDb, err := database.InitMysqlDb(config.MessageDb)
	if err != nil {
		return err
	}
	err = messageDb.AutoMigrate(&database.Group{}, &database.MessageContent{})
	if err != nil {
		return err
	}
	if config.NodeID == 0 {
		config.NodeID = int64(HashCode(config.ServiceID))
	}
	idgen, err := database.NewIDGenerator(config.NodeID)
	if err != nil {
		return err
	}

	rdb, err := conf.InitRedis(config.RedisAddrs, "")
	if err != nil {
		return err
	}

	ns, err := consul.NewNaming(config.ConsulURL)
	if err != nil {
		return err
	}

	ns.Register(&naming.DefaultService{
		Id:       config.ServiceID,
		Name:     wire.SNService,
		Address:  config.PublicAddress,
		Port:     config.PublicPort,
		Protocol: "http",
		Tags:     config.Tags,
		Meta: map[string]string{
			consul.KeyHealthURL: fmt.Sprintf("http://%s:%d/health", config.PublicAddress, config.PublicPort),
		},
	})
	//注销服务
	defer func() {
		ns.Deregister(config.ServiceID)
	}()
	serviceHandler := handler.ServiceHandler{
		BaseDb:    db,
		MessageDb: messageDb,
		Idgen:     idgen,
		Cache:     rdb,
	}
	ac := conf.MakeAccessLog()
	defer ac.Close()

	app := newApp(&serviceHandler)
	app.UseRouter(ac.Handler)
	app.UseRouter(setAllowedResponses)
	return app.Listen(config.Listen, iris.WithOptimizations)
}

func setAllowedResponses(ctx iris.Context) {
	// Indicate that the Server can send JSON and Protobuf for this request.
	ctx.Negotiation().JSON().Protobuf()

	//If client is missing an "Accept: " header then default it to JSON.
	ctx.Negotiation().Accept.JSON()
	ctx.Next()
}

func newApp(serviceHandler *handler.ServiceHandler) *iris.Application {
	app := iris.Default()
	app.Get("/health", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	messageAPI := app.Party("api/:app/message")
	{
		messageAPI.Post("/user", serviceHandler.InsertUserMessage)
		messageAPI.Post("/group", serviceHandler.InsertGroupMessage)
		messageAPI.Post("/ack", serviceHandler.MessageAck)
	}

	groupAPI := app.Party("/api/:app/group")
	{
		groupAPI.Get("/:id", serviceHandler.GroupGet)
		groupAPI.Post("", serviceHandler.GroupCreate)
		groupAPI.Post("/member", serviceHandler.GroupJoin)
		groupAPI.Delete("/member", serviceHandler.GroupQuit)
		groupAPI.Get("/members/:id", serviceHandler.GroupMembers)
	}

	offlineAPI := app.Party("/api/:app/offline")
	{
		offlineAPI.Use(iris.Compression)
		offlineAPI.Post("/index", serviceHandler.GetOfflineMessageIndex)
		offlineAPI.Post("/content", serviceHandler.GetOfflineMessageContent)
	}

	return app
}

func HashCode(key string) uint32 {
	hash32 := crc32.NewIEEE()
	hash32.Write([]byte(key))
	return hash32.Sum32() % 1000
}
