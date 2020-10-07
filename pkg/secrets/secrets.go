package secrets

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"log"
	"os"
)

type SecretsManagerClient interface {
	GetSecretValue(input *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error)
}

type AWSSecrets struct {
	Client SecretsManagerClient
}

func New() (*AWSSecrets, error) {
	currentRegion := os.Getenv("AWS_REGION")
	Client := newSecretsManagerClient(currentRegion)

	return &AWSSecrets{
		Client: Client,
	}, nil
}

func newSecretsManagerClient(region string) SecretsManagerClient {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	return secretsmanager.New(sess)
}

func (s *AWSSecrets) Get(secretname string, region string) (*secretsmanager.GetSecretValueOutput, error) {
	log.Printf("Retrieving %s in %s\n", secretname, region)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretname),
	}

	result, err := s.Client.GetSecretValue(input)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve %s in %s: %w", secretname, region, err)
	}

	return result, nil
}
