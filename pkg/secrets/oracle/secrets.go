package oracle

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"secrets-init/pkg/secrets" //nolint:gci

	log "github.com/sirupsen/logrus"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/example/helpers"
	ocisecret "github.com/oracle/oci-go-sdk/v65/secrets"
	"github.com/pkg/errors"
)

type SecretsProvider struct {
	sm ocisecret.SecretsClient
}

// NewOracleVaultProvider init Google Secrets Provider
func NewOracleVaultProvider(ctx context.Context) (secrets.Provider, error) {
	sp := SecretsProvider{}
	var err error

	log.Info("Criando novo Provider OCI")
	provider, err := auth.InstancePrincipalConfigurationProvider()
	helpers.FatalIfError(err)

	sp.sm, err = ocisecret.NewSecretsClientWithConfigurationProvider(provider)

	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize OCI SDK")
	}
	return &sp, nil
}

// ResolveSecrets replaces all passed variables values prefixed with 'oci:vault:'
// by corresponding secrets from OCI Vault
// The secret name should be in the format (optionally with version)
func (sp *SecretsProvider) ResolveSecrets(ctx context.Context, vars []string) ([]string, error) {
	envs := make([]string, 0, len(vars))
	prefix := "oci:vault:"

	for _, env := range vars {
		kv := strings.Split(env, "=")
		key, value := kv[0], kv[1]
		if strings.HasPrefix(value, prefix) {
			// construct valid secret name
			name := strings.TrimPrefix(value, prefix)
			// if no version specified add latest
			// if !strings.Contains(name, "/versions/") {
			// 	name += "/versions/latest"
			// }

			fmt.Printf("Buscando a secret %s", name)
			// Configuring Request to OCI
			req := ocisecret.GetSecretBundleRequest{
				SecretId: common.String(name),
				Stage:    ocisecret.GetSecretBundleStageLatest}

			resp, err := sp.sm.GetSecretBundle(ctx, req)

			if err != nil {
				return vars, errors.Wrap(err, "failed to get secret from OCI Vault")
			}

			var content string
			base64Details, ok := resp.SecretBundleContent.(ocisecret.Base64SecretBundleContentDetails)
			if ok {
				content = *base64Details.Content
			}

			decoded, err := base64.StdEncoding.DecodeString(content)
			if err != nil {
				return vars, errors.Wrap(err, "failed to get secret from OCI Vault")
			}
			fmt.Printf("Valor da secret %s", decoded)
			env = key + "=" + string(decoded)
		}
		envs = append(envs, env)
	}

	return envs, nil
}
