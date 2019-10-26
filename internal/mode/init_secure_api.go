package mode

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/migotom/mt-bulk/internal/clients"
	"github.com/migotom/mt-bulk/internal/entities"
	"go.uber.org/zap"
)

// InitSecureAPI initializes Mikrotik secure API using SSH client.
func InitSecureAPI(ctx context.Context, sugar *zap.SugaredLogger, client clients.Client, job *entities.Job) ([]entities.CommandResult, []string, error) {
	certificatesDirectory, ok := job.Data["keys_directory"]
	if !ok || certificatesDirectory == "" {
		return nil, nil, fmt.Errorf("keys_directory not specified")
	}

	rootDirectory, ok := job.Data["root_directory"]
	if ok && rootDirectory != "" {
		var err error
		certificatesDirectory, err = clients.SecurePathJoin(rootDirectory, certificatesDirectory)
		if err != nil {
			return nil, nil, err
		}
	}

	results := make([]entities.CommandResult, 0, 9)

	establishResult, err := clients.EstablishConnection(ctx, sugar, client, job)
	results = append(results, establishResult)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	copier, ok := client.(clients.Copier)
	if !ok {
		return nil, nil, fmt.Errorf("copy file operation not implemented for protocol %v", client)
	}

	config := client.GetConfig()

	var sftpCopyResult entities.CommandResult
	sftpCopyResult, err = copier.CopyFile(ctx, filepath.Join(certificatesDirectory, "device.crt"), "sftp://mtbulkdevice.crt")
	results = append(results, sftpCopyResult)
	if err != nil {
		return results, nil, err
	}

	sftpCopyResult, err = copier.CopyFile(ctx, filepath.Join(certificatesDirectory, "device.key"), "sftp://mtbulkdevice.key")
	results = append(results, sftpCopyResult)
	if err != nil {
		return results, nil, err
	}

	// prepare sequence of commands to run on device
	commands := []entities.Command{
		{Body: `/ip service set api-ssl certificate=none`},
		{Body: `/certificate print detail`, MatchPrefix: "c", Match: `(?m)^\s+(\d+).*mtbulkdevice`},
		{Body: `/certificate remove %{c1}`},
		{Body: `/certificate import file-name=mtbulkdevice.crt passphrase=""`, Expect: "certificates-imported: 1"},
		{Body: `/certificate import file-name=mtbulkdevice.key passphrase=""`, Expect: "private-keys-imported: 1", SleepMs: config.VerifySleepMs},
		{Body: `/ip service set api-ssl disabled=no certificate=mtbulkdevice.crt`},
	}

	commandResults, err := clients.ExecuteCommands(ctx, client, commands)
	if err != nil {
		err = fmt.Errorf("executing InitSecureAPI commands error %v", err)
	}
	results = append(results, commandResults...)

	return results, nil, err
}
