// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mgmt

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/siderolabs/siderolink/pkg/agent"
	"github.com/siderolabs/siderolink/pkg/wireguard"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var siderolinkFlags struct {
	joinToken         string
	wireguardEndpoint string
	sinkEndpoint      string
	apiEndpoint       string
	logEndpoint       string
	predefinedPairs   []string
}

var SiderolinkCmd = &cobra.Command{
	Use:    "siderolink-launch",
	Short:  "Internal command used by cluster create to launch siderolink agent",
	Long:   ``,
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer cancel()

		return run(ctx)
	},
}

func init() {
	SiderolinkCmd.PersistentFlags().StringVar(&siderolinkFlags.joinToken, "sidero-link-join-token", "", "join token for the cluster")
	SiderolinkCmd.PersistentFlags().StringVar(&siderolinkFlags.wireguardEndpoint, "sidero-link-wireguard-endpoint", "", "advertised Wireguard endpoint")
	SiderolinkCmd.PersistentFlags().StringVar(&siderolinkFlags.sinkEndpoint, "event-sink-endpoint", "", "gRPC API endpoint for the Event Sink")
	SiderolinkCmd.PersistentFlags().StringVar(&siderolinkFlags.apiEndpoint, "sidero-link-api-endpoint", "", "gRPC API endpoint for the SideroLink")
	SiderolinkCmd.PersistentFlags().StringVar(&siderolinkFlags.logEndpoint, "log-receiver-endpoint", "", "TCP log receiver endpoint")
	SiderolinkCmd.PersistentFlags().StringArrayVar(&siderolinkFlags.predefinedPairs, "predefined-pair", nil, "predefined pairs of UUID=IPv6 addrs for the nodes")

	SiderolinkCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		err := SiderolinkCmd.PersistentFlags().MarkHidden(flag.Name)
		if err != nil {
			panic(err)
		}
	})

	addCommand(SiderolinkCmd)
}

func run(ctx context.Context) error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}

	logger.Info("starting embedded siderolink agent")
	defer logger.Info("stopping embedded siderolink agent")

	err = agent.Run(
		ctx,
		agent.Config{
			WireguardEndpoint: siderolinkFlags.wireguardEndpoint,
			APIEndpoint:       siderolinkFlags.apiEndpoint,
			JoinToken:         siderolinkFlags.joinToken,
			SinkEndpoint:      siderolinkFlags.sinkEndpoint,
			LogEndpoint:       siderolinkFlags.logEndpoint,
			UUIDIPv6Pairs:     siderolinkFlags.predefinedPairs,
			ForceUserspace:    true,
		},
		&handler{l: logger},
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to run siderolink agent: %w", err)
	}

	return nil
}

type handler struct {
	l *zap.Logger
}

func (h *handler) HandlePeerAdded(event wireguard.PeerEvent) error {
	h.l.Info("talos agent sees peer added", zap.String("address", event.Address.String()))

	return nil
}

func (h *handler) HandlePeerRemoved(wgtypes.Key) error {
	return nil
}
