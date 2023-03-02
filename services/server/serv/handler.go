package serv

import (
	"im/logger"

	"github.com/klintcheng/kim/wire"
)

var log = logger.WithFields(logger.Fields{
	"service": wire.SNChat,
	"pkg":     "serv",
})
