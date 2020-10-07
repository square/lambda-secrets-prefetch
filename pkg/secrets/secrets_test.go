package secrets

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/golang/mock/gomock"
)

const (
	validAwsRegion1     = "us-east-1"
	validSecretName1    = "thesecretname"
	validSecretContent1 = "thetestsecret"
)

func newAWSSecret() *AWSSecrets {
	return &AWSSecrets{
		Client: nil,
	}
}

func TestGetSecret(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAWSSecret := newAWSSecret()

	expectedInput := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(validSecretName1),
	}

	sampleGetSecretValueOutput := &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(validSecretContent1),
	}
	mockSecretsManagerClient1 := NewMockSecretsManagerClient(mockCtrl)
	mockSecretsManagerClient1.EXPECT().
		GetSecretValue(expectedInput).
		Times(1).
		DoAndReturn(
			func(input *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
				return sampleGetSecretValueOutput, nil
			})
	mockSecretsManagerClient2 := NewMockSecretsManagerClient(mockCtrl)
	mockSecretsManagerClient2.EXPECT().
		GetSecretValue(gomock.Any()).
		Times(0)

	mockAWSSecret.Client = mockSecretsManagerClient1

	retrievedSecret, err := mockAWSSecret.Get(validSecretName1, validAwsRegion1)
	if err != nil {
		t.Errorf("Failed retrieving a secret's value: %q", err)
	}

	if *retrievedSecret.SecretString != validSecretContent1 {
		t.Error("Incorrect value retrieved from secret")
	}
}
