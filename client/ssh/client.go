package ssh

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/remdit/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	srvConf    config.Server
	conf       *ssh.ClientConfig
	ctx        context.Context
	filepath   string
	client     *ssh.Client
	globalReqs <-chan *ssh.Request
}

func NewClient(ctx context.Context, serverConf config.Server, filepath string) *Client {
	return &Client{
		srvConf:  serverConf,
		ctx:      ctx,
		filepath: filepath,
	}
}

func (c *Client) Connect() error {
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate ed25519 key pair: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to create signer from private key: %w", err)
	}

	c.conf = &ssh.ClientConfig{
		User: fmt.Sprintf("remdit-gocli-%s-%s", runtime.GOOS, runtime.GOARCH),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
			ssh.Password(c.srvConf.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(c.ctx, "tcp", c.srvConf.Addr)
	if err != nil {
		return err
	}
	sshconn, chans, reqs, err := ssh.NewClientConn(conn, c.srvConf.Addr, c.conf)
	if err != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	c.globalReqs = reqs
	c.client = ssh.NewClient(sshconn, chans, reqs)
	return nil
}

func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Client) UploadFile() error {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("failed to create sftp client: %w", err)
	}
	defer sftpClient.Close()

	localFile, err := os.Open(c.filepath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", c.filepath, err)
	}
	defer localFile.Close()

	remoteFilePath := filepath.Base(c.filepath)
	remoteFile, err := sftpClient.Create(remoteFilePath)

	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remoteFilePath, err)
	}
	defer remoteFile.Close()

	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	return nil
}

func (c *Client) GetUploadedFileInfo() (*FileInfoPayload, error) {
	ok, data, err := c.client.SendRequest("file-info", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send file info request: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("failed to get file info: %v", data)
	}
	var fileInfo FileInfoPayload
	if err := ssh.Unmarshal(data, &fileInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file info: %w", err)
	}
	return &fileInfo, nil
}

func (c *Client) ListenServer() error {
	ok, _, err := c.client.SendRequest("listen", true, nil)
	if err != nil {
		return fmt.Errorf("failed to send listen request: %w", err)
	}
	if !ok {
		return errors.New("server did not accept listen request")
	}
	logger := log.FromContext(c.ctx)
	for {
		select {
		case <-c.ctx.Done():
			logger.Info("context done, stopping server listener")
			return nil
		case req, ok := <-c.globalReqs:
			if !ok {
				logger.Warn("server connection closed, stopping server listener")
				return nil
			}
			if req == nil {
				continue
			}
			switch req.Type {
			case "file-save":
				fileContent := req.Payload
				if err := os.WriteFile(c.filepath, fileContent, os.ModePerm); err != nil {
					logger.Error("failed to save file", "error", err)
					req.Reply(false, []byte("failed to save file"))
					continue
				}
				req.Reply(true, []byte("file saved successfully"))
				logger.Infof("file saved successfully with size: %d bytes", len(fileContent))
			default:
				if req.WantReply {
					req.Reply(false, []byte("unknown request type"))
				}
			}
		}
	}
}
