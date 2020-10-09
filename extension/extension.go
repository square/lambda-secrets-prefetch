package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"lambda-secrets-prefetch/pkg/secrets"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

var baseURL string
var client *http.Client
var identifier string

type Config struct {
	SecretsHome    string           `yaml:SecretsHome`
	SecretManagers []SecretManagers `yaml:"SecretManagers"`
}

type SecretManagers struct {
	Prefix  string    `yaml:prefix`
	Secrets []Secrets `yaml:Secrets`
}

type Secrets struct {
	Secretname string `yaml:secretname`
	Filename   string `yaml:filename`
}

// SetLogLevelFromEnv reads the LOG_LEVEL environtment variable for DEBUG or TRACE and updates log verbosity accordingly
func SetLogLevelFromEnv() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		switch strings.ToLower(logLevel) {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "trace":
			log.SetLevel(log.TraceLevel)
		}
	}
}

func getConfig(filename string) *Config {
	var config Config
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(source, &config)
	if err != nil {
		panic(err)
	}

	if config.SecretsHome == "" {
		config.SecretsHome = "/tmp/secrets"
	}

	for i := range config.SecretManagers {
		for j := range config.SecretManagers[i].Secrets {
			if config.SecretManagers[i].Secrets[j].Filename == "" {
				config.SecretManagers[i].Secrets[j].Filename = config.SecretManagers[i].Secrets[j].Secretname
			}
		}
	}

	return &config
}

// This function is only invoked on cold starts

func main() {
	SetLogLevelFromEnv()
	baseURL = fmt.Sprintf("http://%s", os.Getenv("AWS_LAMBDA_RUNTIME_API"))
	log.Infof("[extension] got base url %s", baseURL)
	client = &http.Client{}

	config := getConfig("/var/task/config.yaml")

	// Fetching from secrets manager and writing to disk
	populateSecrets(config)

	var err error
	identifier, err = register()
	if err != nil {
		panic(err)
	}

	next(identifier)
}

func populateSecrets(config *Config) {

	if _, err := os.Stat(config.SecretsHome); os.IsNotExist(err) {
		err := os.Mkdir(config.SecretsHome, 0755)
		if err != nil {
			log.Errorf("[Extension] Error creating directory: %v", err)
		}
	}

	errchan := make(chan error)

	awsSecretsManager, err := secrets.New()
	if err != nil {
		log.Errorf("[Extension] Unable to initiliaze AWSSecrets: %v", err)
	}

	for i := range config.SecretManagers {
		for j := range config.SecretManagers[i].Secrets {
			secretid := fmt.Sprintf("%s%s", config.SecretManagers[i].Prefix, config.SecretManagers[i].Secrets[j].Secretname)
			go handleSecret(*awsSecretsManager, secretid, config.SecretManagers[i].Secrets[j].Filename, config, errchan)
		}
	}

	for i := range config.SecretManagers {
		for range config.SecretManagers[i].Secrets {
			errresult := <-errchan
			if errresult != nil {
				log.Errorf("[Extension]: %v", errresult)
			}
		}
	}

	close(errchan)
}

func handleSecret(sm secrets.AWSSecrets, secretid string, filename string, config *Config, errchan chan<- error) {
	thesecret, err := getSecret(sm, secretid)
	if err != nil {
		errchan <- fmt.Errorf("[Extension] ERROR reading secret (%s): %v", secretid, err)
		return
	}

	err = putSecret(thesecret, filename, config)
	errchan <- err
}

func register() (string, error) {
	data := []byte(`{ "events":["INVOKE", "SHUTDOWN"] }`)
	filename := filepath.Base(os.Args[0])
	log.Infof("[extension] sending name as %s", filename)
	req, err := http.NewRequest("POST", fullURL("2020-01-01/extension/register"), bytes.NewBuffer(data))
	if err != nil {
		log.Fatal("[extension] Error reading request. ", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Lambda-Extension-Name", filename)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error reading response. ", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("[extension] Error reading body. ", err)
		return "", err
	}

	log.Infof("%s", body)

	identifier := resp.Header.Get("Lambda-Extension-Identifier")
	if len(identifier) > 0 {
		log.Infof("[extension] Got extension id %s", identifier)
		return identifier, nil
	}

	return "", fmt.Errorf("Did not get identifier")
}

func next(identifier string) {
	for {
		req, err := http.NewRequest("GET", fullURL("2020-01-01/extension/event/next"), nil)
		if err != nil {
			log.Fatal("[extension] Error reading request. ", err)
		}

		req.Header.Set("Lambda-Extension-Identifier", identifier)

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("[extension] Error reading response. ", err)
			return
		}

		if resp.StatusCode == 500 {
			log.Info("[extension] ... EXITING ...")
			os.Exit(0)
		} else if resp.StatusCode == 403 {
			log.Info("[extension] ... FORBIDDEN ...")
			os.Exit(0)
		} else {
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal("[extension] Error reading body. ", err)
				return
			}

			log.Infof("[extension] %s", body)
		}
	}
}

func fullURL(path string) string {
	log.Infof("[extension] creating new url with base %s and path %s", baseURL, path)
	return fmt.Sprintf("%s/%s", baseURL, path)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func putSecret(secret *string, target string, config *Config) error {
	fname := fmt.Sprintf("%v/%v", config.SecretsHome, target)

	if secret == nil {
		log.Infof("[Extension] Secret %v is unassigned", secret)
		return nil
	}
	log.Infof("[extension] Writing %v", fname)
	d1 := []byte(*secret)
	err := ioutil.WriteFile(fname, d1, 0644)
	if err != nil {
		log.Errorf("[extension] Writing secret error: %v", err)
		return err
	}
	return nil
}

func getSecret(sm secrets.AWSSecrets, secretid string) (*string, error) {

	result, err := sm.Get(secretid, os.Getenv("AWS_REGION"))

	if err != nil {
		return nil, err
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	if result.SecretString != nil {
		secretString = *result.SecretString
		return &secretString, nil
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			log.Error("Base64 Decode Error:", err)
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])
		return &decodedBinarySecret, nil
	}
}
