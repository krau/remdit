package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/krau/remdit/client/ssh"
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

	client := ssh.NewClient(ctx, selectedServer, fp)
	if err := client.Connect(); err != nil {
		logger.Error("failed to connect to server", "addr", selectedServer.Addr, "error", err)
		return
	}
	defer client.Close()
	logger.Debug("connected to server", "addr", selectedServer.Addr)
	if err := client.UploadFile(); err != nil {
		logger.Error("failed to upload file to server", "addr", selectedServer.Addr, "error", err)
		return
	}
	logger.Debug("file uploaded successfully", "filepath", fp)
	fileinfo, err := client.GetUploadedFileInfo()
	if err != nil {
		logger.Error("failed to get file info", "error", err)
		return
	}
	logger.Debugf("file info retrieved: %v", fileinfo)
	logger.Infof("Edit URL: %s\n\n", fileinfo.EditUrl)
	logger.Debug("listening for server events")
	if err := client.ListenServer(); err != nil {
		logger.Error("failed to listen for server events", "error", err)
		return
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
