/*
 * Copyright 2023 The Kubernetes Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"

	"github.com/urfave/cli/v2"

	"github.com/Tal-or/dra-cpu-driver/pkg/config"
	"github.com/Tal-or/dra-cpu-driver/pkg/driver"
	"github.com/Tal-or/dra-cpu-driver/pkg/flags"
)

func main() {
	if err := newApp().Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newApp() *cli.App {
	progArgs := &config.ProgArgs{
		LoggingConfig: flags.NewLoggingConfig(),
	}
	cliFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "node-name",
			Usage:       "The name of the node to be worked on.",
			Required:    true,
			Destination: &progArgs.NodeName,
			EnvVars:     []string{"NODE_NAME"},
		},
		&cli.StringFlag{
			Name:        "cdi-root",
			Usage:       "Absolute path to the directory where CDI files will be generated.",
			Value:       "/etc/cdi",
			Destination: &progArgs.CdiRoot,
			EnvVars:     []string{"CDI_ROOT"},
		},
		&cli.StringFlag{
			Name:        "reserved-cpus",
			Usage:       "reserved-cpus",
			Value:       "",
			Destination: &progArgs.Reserved,
			EnvVars:     []string{"RESERVED_CPUS"},
		},
		&cli.StringFlag{
			Name:        "allocatable-cpus",
			Usage:       "allocatable-cpus",
			Value:       "",
			Destination: &progArgs.Allocatable,
			EnvVars:     []string{"ALLOCATABLE_CPUS"},
		},
		&cli.StringFlag{
			Name:        "shared-cpus",
			Usage:       "shared-cpus",
			Value:       "",
			Destination: &progArgs.Shared,
			EnvVars:     []string{"SHARED_CPUS"},
		},
	}
	cliFlags = append(cliFlags, progArgs.KubeClientConfig.Flags()...)
	cliFlags = append(cliFlags, progArgs.LoggingConfig.Flags()...)

	app := &cli.App{
		Name:            "dra-cpu-kubeletplugin",
		Usage:           "dra-cpu-kubeletplugin implements a DRA driver plugin.",
		ArgsUsage:       " ",
		HideHelpCommand: true,
		Flags:           cliFlags,
		Before: func(c *cli.Context) error {
			if c.Args().Len() > 0 {
				return fmt.Errorf("arguments not supported: %v", c.Args().Slice())
			}
			return progArgs.LoggingConfig.Apply()
		},
		Action: func(c *cli.Context) error {
			ctx := c.Context
			clientSets, err := progArgs.KubeClientConfig.NewClientSets()
			if err != nil {
				return fmt.Errorf("create client: %v", err)
			}

			cfg := &config.Config{
				ProgArgs:   progArgs,
				Coreclient: clientSets.Core,
			}

			return StartPlugin(ctx, cfg)
		},
	}

	return app
}

func StartPlugin(ctx context.Context, cfg *config.Config) error {
	err := os.MkdirAll(config.DriverPluginPath, 0750)
	if err != nil {
		return err
	}

	drv, err := driver.New(ctx, cfg)
	if err != nil {
		return err
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigc

	err = drv.Shutdown(ctx)
	if err != nil {
		klog.FromContext(ctx).Error(err, "Unable to cleanly shutdown driver")
	}

	return nil
}
