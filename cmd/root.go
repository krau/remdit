package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/coder/websocket"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/krau/remdit/client"
	"github.com/krau/remdit/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "remdit",
	Short:   "A collaborative text editor for remote files",
	Example: "remdit file.json",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("version") {
			fmt.Println("Remdit Version:", config.Version)
			fmt.Println("Commit:", config.Commit)
			os.Exit(0)
		}
		if cmd.Flags().Changed("upgrade") {
			if err := Upgrade(); err != nil {
				fmt.Println("Failed to upgrade:", err)
				os.Exit(1)
			}
		}
		if len(args) == 0 {
			return nil
		}
		logger := log.Default()
		logger.SetTimeFormat("")
		logger.SetReportTimestamp(false)
		logger.SetReportCaller(false)
		if cmd.Flags().Changed("verbose") {
			logger.SetLevel(log.DebugLevel)
		}
		ctx := log.WithContext(cmd.Context(), logger)
		cmd.SetContext(ctx)

		err := config.LoadConfig(cmd.Context())
		if err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Usage()
		}
		fp := args[0]
		if !fileutil.IsExist(fp) {
			return os.ErrNotExist
		}
		if fileutil.IsDir(fp) {
			return fmt.Errorf("%s is a directory, not a file", fp)
		}
		absFp, err := filepath.Abs(fp)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		ctx := cmd.Context()
		run(ctx, absFp)
		return nil
	},
}

func init() {
	rootCmd.Flags().BoolP("verbose", "v", false, "enable verbose output")
	rootCmd.Flags().BoolP("version", "V", false, "print version information")
	rootCmd.Flags().BoolP("upgrade", "u", false, "upgrade remdit to the latest version")
}

func run(ctx context.Context, fp string) {
	logger := log.FromContext(ctx)
	servers := config.C.Servers
	if len(servers) == 0 {
		logger.Error("no servers configured")
		return
	}
	// random choose a server
	var validServers []config.Server
	for _, server := range servers {
		if server.Valid() {
			validServers = append(validServers, server)
		}
	}
	if len(validServers) == 0 {
		logger.Error("no valid servers found")
		return
	}
	selectedServer := validServers[rand.Intn(len(validServers))]
	logger.Debug("selected server", "addr", selectedServer.Addr)

	client := client.NewClient(ctx, selectedServer, fp)
	if err := client.CreateSession(); err != nil {
		logger.Error("failed to create session", "error", err)
		return
	}
	if err := client.Connect(); err != nil {
		logger.Error("failed to connect to server", "addr", selectedServer.Addr, "error", err)
		return
	}
	logger.Debug("connected to server", "addr", selectedServer.Addr)
	editUrl := client.GetEditURL()
	logger.Infof("Edit URL for file %s: %s\nDO NOT SHARE TO STRANGERS!", filepath.Base(fp), editUrl)

	err := client.HandleMessages()
	if err != nil {
		logger.Error("error handling messages", "error", err)
		if closeErr := client.Close(websocket.StatusInternalError, err.Error()); closeErr != nil {
			logger.Error("failed to close connection", "error", closeErr)
		}
		return
	}
	logger.Debug("session ended")
	if err := client.Close(websocket.StatusNormalClosure, "session ended"); err != nil {
		logger.Error("failed to close connection", "error", err)
	}
}

func Execute() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.FromContext(ctx).Error(err)
		return err
	}
	return nil
}
