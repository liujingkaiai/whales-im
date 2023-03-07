package main

import (
	"context"
	"flag"
	"im/logger"
	"im/services/gateway"
	"im/services/server"

	"github.com/spf13/cobra"
)

const version = "v1"

func main() {
	flag.Parse()

	root := &cobra.Command{
		Use:     "im",
		Version: version,
		Short:   "IM Cloud",
	}
	ctx := context.Background()
	root.AddCommand(gateway.NewServerStartCmd(ctx, root.Version))
	root.AddCommand(server.NewServerStartCmd(ctx, root.Version))
	if err := root.Execute(); err != nil {
		logger.WithError(err).Fatal("Could not run command")
	}
}
